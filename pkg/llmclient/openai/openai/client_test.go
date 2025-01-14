package openai

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/y0ug/ai-helper/pkg/llmclient/openai/middleware"
	"github.com/y0ug/ai-helper/pkg/llmclient/openai/requestoption"
)

func skipIfNoAPIKey(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("Skipping integration test because OPENAI_API_KEY is not set")
	}
}

func TestClientStreamIntegration(t *testing.T) {
	skipIfNoAPIKey(t)

	client := NewClient()
	// requestoption.WithMiddleware(middleware.LoggingMiddleware()))
	ctx := context.Background()

	t.Run("ChatCompletion", func(t *testing.T) {
		params := ChatCompletionNewParams{
			Model: "gpt-3.5-turbo",
			Messages: []ChatCompletionMessageParam{
				{
					Role:    "user",
					Content: "Write a 2048 word essay on the topic of artificial intelligence",
				},
			},
			Temperature: 0,
		}
		stream := client.Chat.NewStreaming(ctx, params)
		for stream.Next() {
			evt := stream.Current()
			if len(evt.Choices) == 0 {
				continue
			}
			print(fmt.Sprintf("%s", evt.Choices[0].Delta.Content))
		}
	})
}

func TestClientIntegration(t *testing.T) {
	skipIfNoAPIKey(t)

	client := NewClient(
		requestoption.WithMiddleware(middleware.LoggingMiddleware()))
	ctx := context.Background()

	t.Run("ChatCompletion", func(t *testing.T) {
		params := ChatCompletionNewParams{
			Model: "gpt-3.5-turbo",
			Messages: []ChatCompletionMessageParam{
				{
					Role:    "user",
					Content: "Say hello in exactly 5 words",
				},
			},
			Temperature: 0,
		}

		completion, err := client.Chat.New(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create chat completion: %v", err)
		}

		if len(completion.Choices) == 0 {
			t.Fatal("Expected at least one choice in response")
		}

		if completion.Model == "" {
			t.Error("Expected model to be set in response")
		}

		if completion.Usage.TotalTokens == 0 {
			t.Error("Expected non-zero token usage")
		}

		rateLimit := client.GetRateLimit()
		t.Logf("Rate limit: %+v", rateLimit)
		t.Logf("Chat completion: %v", completion.Choices[0].Message.Content)
	})

	t.Run("ChatCompletionWithTimeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		params := ChatCompletionNewParams{
			Model: "gpt-3.5-turbo",
			Messages: []ChatCompletionMessageParam{
				{
					Role:    "user",
					Content: "Write a very long essay about artificial intelligence",
				},
			},
		}

		_, err := client.Chat.New(ctx, params)
		if err == nil {
			t.Error("Expected timeout error but got none")
		}
	})

	t.Run("InvalidModel", func(t *testing.T) {
		params := ChatCompletionNewParams{
			Model: "non-existent-model",
			Messages: []ChatCompletionMessageParam{
				{
					Role:    "user",
					Content: "Hello",
				},
			},
		}

		_, err := client.Chat.New(ctx, params)
		if err == nil {
			t.Error("Expected error for invalid model but got none")
		}
	})
}
