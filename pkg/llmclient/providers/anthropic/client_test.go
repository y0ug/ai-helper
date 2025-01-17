package anthropic

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/y0ug/ai-helper/pkg/llmclient/common"
)

func skipIfNoAPIKey(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("Skipping integration test because ANTHROPIC_API_KEY is not set")
	}
}

func TestClientStreamIntegration(t *testing.T) {
	skipIfNoAPIKey(t)

	client := NewClient()
	ctx := context.Background()

	t.Run("ChatCompletion", func(t *testing.T) {
		params := MessageNewParams{
			Model:     "claude-3-5-sonnet-20241022",
			MaxTokens: 4096,
			Messages: []MessageParam{
				{
					Role: "user",
					Content: []*common.AIContent{common.NewTextContent(
						"Write a 100 word essay on the topic of artificial intelligence",
					)},
				},
			},
			Temperature: 0,
		}
		stream, err := client.Message.NewStreaming(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create chat completion stream: %v", err)
		}
		message := Message{}
		for stream.Next() {
			evt := stream.Current()
			message.Accumulate(evt)
			// evt := stream.Current()
			// switch evt := evt.(type) {
			// case *MessageStreamEvent:
			// 	message.Accumulate(*evt)
			// 	// fmt.Printf("%s ", evt.Type)
			// 	switch evt.Type {
			// 	case "message_start":
			// 	case "content_block_start":
			// 	case "content_block_delta":
			// 		// fmt.Printf("%v\n", evt.ContentBlock)
			// 		// fmt.Printf("Content: %v\n", evt.Delta)
			// 		fmt.Printf("%s", evt.Delta)
			// 	case "content_block_stop":
			// 	case "message_delta":
			// 		// fmt.Printf("%v\n", evt.ContentBlock)
			// 		// fmt.Printf("Content: %v\n", evt.Delta)
			// 	case "message_stop":
			// 	}
			// 	// fmt.Printf("\n")
			// default:
			// }
		}
		if stream.Err() != nil {
			fmt.Printf("Error: %v\n", stream.Err())
		}

		fmt.Printf("Message: %v\n", message.Content)
	})
}

func TestClientIntegration(t *testing.T) {
	skipIfNoAPIKey(t)

	client := NewClient()
	ctx := context.Background()

	t.Run("ChatCompletion", func(t *testing.T) {
		params := MessageNewParams{
			Model:     "claude-3-5-sonnet-20241022",
			MaxTokens: 4096,
			Messages: []MessageParam{
				{
					Role: "user",
					Content: []*common.AIContent{common.NewTextContent(
						"Say hello in exactly 5 words",
					)},
				},
			},
			Temperature: 0,
		}

		message, err := client.Message.New(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create chat completion: %v", err)
		}

		if len(message.Content) == 0 {
			t.Fatal("Expected at least one choice in response")
		}

		if message.Model == "" {
			t.Error("Expected model to be set in response")
		}

		if message.Usage.InputTokens == 0 {
			t.Error("Expected non-zero token usage")
		}

		t.Logf("Message: %v", message.Content[0])
	})

	t.Run("InvalidModel", func(t *testing.T) {
		params := MessageNewParams{
			Model:     "non-existent-model",
			MaxTokens: 4096,
			Messages: []MessageParam{
				{
					Role: "user",
					Content: []*common.AIContent{common.NewTextContent(
						"Say hello in exactly 5 words",
					)},
				},
			},
		}

		_, err := client.Message.New(ctx, params)
		if err == nil {
			t.Error("Expected error for invalid model but got none")
		}
	})
}
