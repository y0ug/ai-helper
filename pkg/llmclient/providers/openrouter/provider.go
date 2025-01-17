package openrouter

import (
	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/openai"
	"github.com/y0ug/ai-helper/pkg/llmclient/types"
)

type OpenRouterProvider struct {
	*openai.OpenAIProvider
}

func NewOpenRouterProvider(opts ...options.RequestOption) types.LLMProvider {
	return &OpenRouterProvider{
		&openai.OpenAIProvider{
			Client: NewClient(opts...).Client,
		},
	}
}
