package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/openai"
)

func highlightAndPrint(
	w *bufio.Writer,
	line string,
	style *chroma.Style,
	lexer chroma.Lexer,
	formatter chroma.Formatter,
) {
	iterator, err := lexer.Tokenise(nil, line)
	if err != nil {
		log.Printf("Tokenization error: %v", err)
		return
	}

	// var buf bytes.Buffer
	err = formatter.Format(w, style, iterator)
	if err != nil {
		log.Printf("Formatting error: %v", err)
		return
	}
}

func main() {
	client := openai.NewClient()
	// requestoption.WithMiddleware(middleware.LoggingMiddleware()))
	ctx := context.Background()
	params := openai.ChatCompletionNewParams{
		Model: "gpt-4o",
		Messages: []openai.ChatCompletionMessageParam{
			{
				Role:    "user",
				Content: "Can you write a 10 lines, Go code? You will prefix the code in fenced code block with the language name",
			},
		},
		Temperature: 0,
	}
	stream := client.Chat.NewStreaming(ctx, params)

	// Initialize Chroma lexer and formatter
	lexer := lexers.Get("markdown")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	defaultLexer := lexer

	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// State to handle multi-line constructs (e.g., code blocks)
	inCodeBlock := false

	// Buffered writer for efficient output
	// _ = bufio.NewWriterSize(nil, 0)

	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	var buffer bytes.Buffer

	style := styles.Get("monokai")
	if style == nil {
		fmt.Println("style not found")
		style = styles.Fallback
	}
	for stream.Next() {
		evt := stream.Current()
		if len(evt.Choices) == 0 {
			continue
		}
		content := evt.Choices[0].Delta.Content
		if content == "" {
			continue
		}
		// fmt.Printf("content: \"%s\" ", content)
		// highlightAndPrint(w, content, style, lexer, formatter)

		// Split the incoming content into lines
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

			if strings.HasPrefix(line, "```") {
				inCodeBlock = !inCodeBlock
				if inCodeBlock {
					line = strings.ToLower(line) // ChatGPT put a Uppercase sometimes
					highlightAndPrint(w, line, style, lexer, formatter)
					language := strings.Trim(strings.ToLower(line[3:]), "\n")
					// fmt.Printf("Language: %s\n", language)
					lexer = lexers.Get(language)
					if lexer == nil {
						lexer = defaultLexer
					}
					continue
				} else {
					lexer = defaultLexer
				}
			}
			highlightAndPrint(w, line, style, lexer, formatter)
		}
	}
	highlightAndPrint(w, buffer.String(), style, lexer, formatter)

	if err := stream.Err(); err != nil {
		log.Fatalf("Stream error: %v", err)
	}
}
