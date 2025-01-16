package llmclient

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/y0ug/ai-helper/pkg/llmclient/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/deepseek"
	"github.com/y0ug/ai-helper/pkg/llmclient/gemini"
	"github.com/y0ug/ai-helper/pkg/llmclient/openrouter"
)

func TestOtherProvider_Send(t *testing.T) {
	test := []struct {
		Provider common.LLMProvider
		Model    string
	}{
		{
			&OpenRouterProvider{
				&OpenAIProvider{
					client: openrouter.NewClient().Client,
				},
			},
			"gpt-3.5-turbo",
		},
		{
			&GeminiProvider{
				&OpenAIProvider{
					client: gemini.NewClient().Client,
				},
			},
			"gemini-exp-1206",
		},
		{
			&DeepseekProvider{
				client: deepseek.NewClient(),
			},
			"deepseek-chat",
		},
	}

	for _, v := range test {
		t.Run(fmt.Sprintf("Provider %T", v.Provider), func(t *testing.T) {
			ctx := context.Background()
			params := common.ChatMessageNewParams{
				Model:       v.Model,
				MaxTokens:   100,
				Temperature: 0.7,
				Messages: []common.ChatMessageParams{
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

			if !assert.NoError(t, err) {
				t.FailNow()
			}
			assert.NotNil(t, response)
			// assert.NotEmpty(t, response.ID) // Gemini does not return ID
			assert.NotEmpty(t, response.Model)
			assert.Greater(t, response.Usage.InputTokens, 0)
			assert.Greater(t, response.Usage.OutputTokens, 0)
			fmt.Println(response.Choice[0].Content[0].String())
		})
	}
}
