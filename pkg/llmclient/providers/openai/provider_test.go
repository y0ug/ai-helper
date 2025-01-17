package openai

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/y0ug/ai-helper/pkg/llmclient/types"
)

func TestFromLLMMessageToOpenAi(t *testing.T) {
	tests := []struct {
		name     string
		messages []*types.ChatMessage
		want     []ChatCompletionMessageParam
	}{
		{
			name: "basic text message",
			messages: []*types.ChatMessage{
				{
					Role: "user",
					Content: []*types.MessageContent{
						types.NewTextContent("Hello"),
					},
				},
			},
			want: []ChatCompletionMessageParam{
				{
					Role:    "user",
					Content: "Hello",
				},
			},
		},
		{
			name: "tool result message",
			messages: []*types.ChatMessage{
				{
					Role: "tool",
					Content: []*types.MessageContent{
						{
							Type:      types.ContentTypeToolResult,
							Content:   "Result",
							ToolUseID: "tool123",
						},
					},
				},
			},
			want: []ChatCompletionMessageParam{
				{
					Role:       "tool",
					Content:    "Result",
					ToolCallID: "tool123",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MessageToOpenAI(tt.messages...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOpenAIProvider_Send(t *testing.T) {
	// Create a new adapter with a mock client
	adapter := NewOpenAIProvider()

	ctx := context.Background()
	params := types.ChatParams{
		Model:       "gpt-3.5-turbo",
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

	// This is an integration test that requires an actual OpenAI API key
	// You might want to skip it if no API key is present
	// t.Skip("Skipping integration test - requires OpenAI API key")

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
