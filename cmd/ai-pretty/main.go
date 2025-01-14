package main

import (
	"bufio"
	"context"
	"log"
	"os"

	"github.com/y0ug/ai-helper/pkg/highlighter"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/common"
)

func main() {
	// const model = "claude-3-5-sonnet-20241022"
	const model = "gpt-4o"
	provider, _ := llmclient.NewProviderByModel(model, nil)

	ctx := context.Background()
	params := common.BaseChatMessageNewParams{
		Model:       model,
		MaxTokens:   1024,
		Temperature: 0,
		Messages: llmclient.NewMessagesParams(
			llmclient.NewUserMessage(
				"Write a 1000 word essai about Golang and put a some code block in the middle",
			),
		),
	}

	stream := provider.Stream(ctx, params)
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
