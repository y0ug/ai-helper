package anthropic

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/y0ug/ai-helper/pkg/llmclient/types"
)

func TestAnthropicProvider_Send(t *testing.T) {
	// Create a new adapter with a client
	adapter := &AnthropicProvider{
		client: NewClient(),
	}

	ctx := context.Background()
	params := types.ChatParams{
		Model:       "claude-3-opus-20240229",
		MaxTokens:   100,
		Temperature: 0.7,
		Messages: []*types.ChatMessage{
			{
				Role: "system",
				Content: []*types.MessageContent{
					types.NewTextContent("You are a helpful AI assistant."),
				},
			},
			{
				Role: "user",
				Content: []*types.MessageContent{
					types.NewTextContent("Hello, how are you?"),
				},
			},
		},
	}

	// This is an integration test that requires an actual Anthropic API key
	// You might want to skip it if no API key is present
	// t.Skip("Skipping integration test - requires Anthropic API key")

	response, err := adapter.Send(ctx, params)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.ID)
	assert.NotEmpty(t, response.Model)

	// Check if we have choices and content before accessing them
	if assert.NotEmpty(t, response.Choice, "Should have at least one choice") {
		if assert.NotEmpty(t, response.Choice[0].Content, "Choice should have content") {
			content := response.Choice[0].Content[0].String()
			fmt.Printf("Response content: %s\n", content)
			assert.NotEmpty(t, content)
		}
	}

	// Check usage statistics if they exist
	if assert.NotNil(t, response.Usage, "Should have usage statistics") {
		fmt.Printf("Usage - Input tokens: %d, Output tokens: %d\n",
			response.Usage.InputTokens,
			response.Usage.OutputTokens)
		assert.Greater(t, response.Usage.InputTokens, 0)
		assert.Greater(t, response.Usage.OutputTokens, 0)
	}
}
