package openai

import (
	"context"
	"os"
	"testing"
	"time"
)

func skipIfNoAPIKey(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("Skipping integration test because OPENAI_API_KEY is not set")
	}
}

func TestClientIntegration(t *testing.T) {
	skipIfNoAPIKey(t)

	client := NewClient()
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
