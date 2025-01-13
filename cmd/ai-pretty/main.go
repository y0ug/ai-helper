package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/y0ug/ai-helper/pkg/llmclient/openai"
)

func main() {
	client := openai.NewClient()
	// requestoption.WithMiddleware(middleware.LoggingMiddleware()))
	ctx := context.Background()
	params := openai.ChatCompletionNewParams{
		Model: "gpt-3.5-turbo",
		Messages: []openai.ChatCompletionMessageParam{
			{
				Role:    "user",
				Content: "Write a fictional technical documentation in markdown no more then 2048 words",
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
	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// State to handle multi-line constructs (e.g., code blocks)
	inCodeBlock := false

	// Buffered writer for efficient output
	// _ = bufio.NewWriterSize(nil, 0)

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
			// Remove the processed line from the buffer
			buffer.Next(index + 1)

			// Handle code block state
			if strings.HasPrefix(line, "```") {
				inCodeBlock = !inCodeBlock
			}

			// Append the line to the lexer input
			var lineToHighlight string
			if inCodeBlock {
				// If inside a code block, specify the language if provided
				lineToHighlight = line + "\n"
			} else {
				lineToHighlight = line + "\n"
			}

			// Tokenize and format the current line
			iterator, err := lexer.Tokenise(nil, lineToHighlight)
			if err != nil {
				log.Printf("Tokenization error: %v", err)
				continue
			}

			var highlighted strings.Builder
			err = formatter.Format(&highlighted, style, iterator)
			if err != nil {
				log.Printf("Formatting error: %v", err)
				continue
			}

			// Print the highlighted line
			fmt.Print(highlighted.String())

		}
	}
	if err := stream.Err(); err != nil {
		log.Fatalf("Stream error: %v", err)
	}
}
