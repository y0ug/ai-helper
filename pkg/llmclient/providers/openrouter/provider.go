package openrouter

import (
	"github.com/y0ug/ai-helper/pkg/llmclient/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/requestoption"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/openai"
)

type OpenRouterProvider struct {
	*openai.OpenAIProvider
}

func NewOpenRouterProvider(opts ...requestoption.RequestOption) common.LLMProvider {
	return &OpenRouterProvider{
		&openai.OpenAIProvider{
			Client: NewClient(opts...).Client,
		},
	}
}
