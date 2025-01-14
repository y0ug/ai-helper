package llmclient

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/anthropic"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/common"
)

func TestAntropicAdapter_Send(t *testing.T) {
	// Create a new adapter with a client
	adapter := &AntropicAdapter{
		client: anthropic.NewClient(),
	}

	ctx := context.Background()
	params := common.BaseChatMessageNewParams{
		Model:       "claude-3-opus-20240229",
		MaxTokens:   100,
		Temperature: 0.7,
		Messages: []common.BaseChatMessageParams{
			{
				Role: "system",
				Content: []*common.AIContent{
					common.NewTextContent("You are a helpful AI assistant."),
				},
			},
			{
				Role: "user",
				Content: []*common.AIContent{
					common.NewTextContent("Hello, how are you?"),
				},
			},
		},
	}

	// This is an integration test that requires an actual Anthropic API key
	// You might want to skip it if no API key is present
	// t.Skip("Skipping integration test - requires Anthropic API key")

	response, err := adapter.Send(ctx, params)

	fmt.Println(response.Choice[0].Content[0].String())
	fmt.Printf("Usage: %d %d\n", response.Usage.InputTokens, response.Usage.OutputTokens)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.ID)
	assert.NotEmpty(t, response.Model)
	assert.Greater(t, response.Usage.InputTokens, 0)
	assert.Greater(t, response.Usage.OutputTokens, 0)
}
