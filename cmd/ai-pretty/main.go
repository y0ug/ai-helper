package main

import (
	"bufio"
	"bytes"
	"context"
	"log"
	"os"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/common"
)

type Highlighter struct {
	w            *bufio.Writer
	InCodeBlock  bool
	Lexer        chroma.Lexer
	defaultLexer chroma.Lexer
	Formatter    chroma.Formatter
	Style        *chroma.Style
}

func (h *Highlighter) ProcessLine(line string) {
	if strings.HasPrefix(line, "```") {
		h.InCodeBlock = !h.InCodeBlock
		if h.InCodeBlock {
			line = strings.ToLower(line) // ChatGPT put a Uppercase sometimes
			h.highlightAndPrint(line)
			language := strings.Trim(strings.ToLower(line[3:]), "\n")
			// fmt.Printf("Language: %s\n", language)
			h.Lexer = lexers.Get(language)
			if h.Lexer == nil {
				h.Lexer = h.defaultLexer
			}
			return
		} else {
			h.Lexer = h.defaultLexer
		}
	}
	h.highlightAndPrint(line)
}

func (h *Highlighter) highlightAndPrint(line string) {
	iterator, err := h.Lexer.Tokenise(nil, line)
	if err != nil {
		log.Printf("Tokenization error: %v", err)
		return
	}

	// var buf bytes.Buffer
	err = h.Formatter.Format(h.w, h.Style, iterator)
	if err != nil {
		log.Printf("Formatting error: %v", err)
		return
	}
}

func ConsumeStream(ch <-chan string) {
	// Initialize Chroma lexer and formatter
	lexer := lexers.Get("markdown")
	if lexer == nil {
		lexer = lexers.Fallback
	}

	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	var buffer bytes.Buffer

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	h := &Highlighter{
		Lexer:        lexer,
		defaultLexer: lexer,
		InCodeBlock:  false,
		Style:        style,
		Formatter:    formatter,
		w:            w,
	}

	for content := range ch {
		buffer.WriteString(content)
		for {
			currentBuffer := buffer.String()
			index := strings.Index(currentBuffer, "\n")
			if index == -1 {
				// No complete line yet
				break
			}
			// Extract the line up to the newline
			line := currentBuffer[:index+1] // Include the newline character
			// fmt.Printf("Line: %s", line)
			// Remove the processed line from the buffer
			buffer.Next(index + 1)
			//

			h.ProcessLine(line)
		}
	}
	remaining := buffer.String()
	if len(remaining) > 0 {
		// Check if the remaining buffer ends with a newline
		if strings.HasSuffix(remaining, "\n") {
			h.ProcessLine(remaining)
		} else {
			// Optionally, handle incomplete lines differently
			// For example, you might choose to ignore them or process them based on context
			// Here, we'll choose to ignore incomplete lines to prevent extra characters
			// Alternatively, you could add a newline before processing
			remaining += "\n"
			h.ProcessLine(remaining)
		}
	}
}

func main() {
	model := "claude-3-5-sonnet-20241022"
	// model := "deepseek-chat"
	// model := "gpt-4o"
	provider, _ := llmclient.NewProviderByModel(model, nil)

	// requestoption.WithMiddleware(middleware.LoggingMiddleware()))
	ctx := context.Background()
	params := common.BaseChatMessageNewParams{
		Model:     model,
		MaxTokens: 1024,
		Messages: []common.BaseChatMessageParams{
			{
				Role: "user",
				Content: []*common.AIContent{
					common.NewTextContent(
						"Write an Hello World in golang, with your model name inside. The code have to be in makdown code fence with the language.",
					),
				},
			},
		},
		Temperature: 0,
	}
	stream := provider.Stream(ctx, params)
	eventCh := make(chan string)

	// Start ConsumeStream in a separate goroutine
	go func() {
		if err := llmclient.ConsumeStream(stream, eventCh); err != nil {
			log.Fatalf("Error consuming stream: %v", err)
		}
	}()

	ConsumeStream(eventCh)
}
