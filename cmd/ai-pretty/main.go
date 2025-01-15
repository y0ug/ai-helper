package main

import (
	"bufio"
	"context"
	"log"
	"os"

	"github.com/y0ug/ai-helper/pkg/highlighter"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestoption"
)

func StrToPtr(s string) *string {
	return &s
}

func main() {
	// const model = "claude-3-5-sonnet-20241022"
	const model = "gpt-4o"
	requestOpts := []requestoption.RequestOption{
		// requestoption.WithMiddleware(middleware.LoggingMiddleware()),
	}
	provider, _ := llmclient.NewProviderByModel(model, nil, requestOpts...)

	ctx := context.Background()
	params := llmclient.NewChatParams(
		llmclient.WithModel(model),
		llmclient.WithMaxTokens(1024),
		llmclient.WithTemperature(0),
		llmclient.WithMessages(
			llmclient.NewUserMessage(
				"What the weather ?",
				// "Write a 1000 word essai about Golang and put a some code block in the middle",
			),
		),
		// llmclient.WithTools(common.Tool{
		// 	Name:        "get_weather",
		// 	Description: StrToPtr("Get the current weather"),
		// 	InputSchema: map[string]interface{}{},
		// },
		// ),
	)

	stream := provider.Stream(ctx, *params)
	eventCh := make(chan string)

	go func() {
		if err := llmclient.ConsumeStream(ctx, stream, eventCh); err != nil {
			if err != context.Canceled {
				log.Printf("Error consuming stream: %v", err)
			}
		}
	}()

	writer := bufio.NewWriter(os.Stdout)
	h := highlighter.NewHighlighter(writer)
	if err := h.ProcessStream(ctx, eventCh); err != nil {
		if err != context.Canceled {
			log.Printf("Error processing stream: %v", err)
		}
	}
}
