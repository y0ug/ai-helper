package main

import (
	"bufio"
	"context"
	"log"
	"os"

	"github.com/y0ug/ai-helper/cmd/ai-pretty/highlighter"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/common"
)

func main() {
	const model = "claude-3-5-sonnet-20241022"
	provider, err := llmclient.NewProviderByModel(model, nil)
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	params := common.BaseChatMessageNewParams{
		Model:       model,
		MaxTokens:   1024,
		Temperature: 0,
		Messages: []common.BaseChatMessageParams{
			{
				Role: "user",
				Content: []*common.AIContent{
					common.NewTextContent(
						"Write an Hello World in golang, with your model name inside. The code have to be in markdown code fence with the language.",
					),
				},
			},
		},
	}

	stream := provider.Stream(ctx, params)
	eventCh := make(chan string)

	go func() {
		if err := llmclient.ConsumeStream(stream, eventCh); err != nil {
			log.Fatalf("Error consuming stream: %v", err)
		}
	}()

	writer := bufio.NewWriter(os.Stdout)
	highlighter.ProcessStream(eventCh, writer)
}
