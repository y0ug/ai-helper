package llmclient

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/gemini"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/openrouter"
)

func TestOtherProvider_Send(t *testing.T) {
	test := []struct {
		Provider common.LLMClient
		Model    string
	}{
		{
			&OpenRouterProvider{
				OpenAIProvider{
					client: &openrouter.NewClient().Client,
				},
			},
			"gpt-3.5-turbo",
		},
		{
			&GeminiProvider{
				OpenAIProvider{
					client: &gemini.NewClient().Client,
				},
			},
			"gemini-pro",
		},
	}

	for _, v := range test {
		ctx := context.Background()
		params := common.BaseChatMessageNewParams{
			Model:       v.Model,
			MaxTokens:   100,
			Temperature: 0.7,
			Messages: []common.BaseChatMessageParams{
				{
					Role: "user",
					Content: []*common.AIContent{
						common.NewTextContent("Hello, how are you?"),
					},
				},
			},
		}

		// This is an integration test that requires an actual OpenAI API key
		// You might want to skip it if no API key is present
		// t.Skip("Skipping integration test - requires OpenAI API key")

		response, err := v.Provider.Send(ctx, params)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.ID)
		assert.NotEmpty(t, response.Model)
		assert.Greater(t, response.Usage.InputTokens, 0)
		assert.Greater(t, response.Usage.OutputTokens, 0)
		fmt.Println(response.Choice[0].Content[0].String())
		fmt.Printf("Usage: %d %d\n", response.Usage.InputTokens, response.Usage.OutputTokens)
	}
}
