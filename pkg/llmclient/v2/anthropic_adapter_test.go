package llmclient

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnthropicAdapter_Integration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=1 to run")
	}

	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	client := NewAnthropicAdapter(apiKey)

	ctx := context.Background()
	req := ChatRequest{
		Messages: []Message{
			{
				Role:    RoleUser,
				Content: "What is 2+2?",
			},
		},
		Model: ModelClaude2,
	}

	resp, err := client.Chat(ctx, req)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Content)
	require.NotEmpty(t, resp.Model)
}
