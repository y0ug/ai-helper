package gemini

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/y0ug/ai-helper/pkg/llmclient/types"
)

func TestSend(t *testing.T) {
	// Create a new provder with a mock client
	provder := NewGeminiProvider()
	ctx := context.Background()
	params := types.ChatParams{
		Model:       "gemini-exp-1206",
		MaxTokens:   100,
		Temperature: 0.7,
		Messages: []*types.ChatMessage{
			{
				Role: "user",
				Content: []*types.MessageContent{
					types.NewTextContent("Hello, how are you?"),
				},
			},
		},
	}

	// t.Skip("Skipping integration test - requires API key")

	response, err := provder.Send(ctx, params)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	// Gemini don't set an response.ID
	// assert.NotEmpty(t, response.ID)
	assert.NotEmpty(t, response.Model)
	assert.Greater(t, response.Usage.InputTokens, 0)
	assert.Greater(t, response.Usage.OutputTokens, 0)
	fmt.Println(response.Choice[0].Content[0].String())
	fmt.Printf("Usage: %d %d\n", response.Usage.InputTokens, response.Usage.OutputTokens)
}
