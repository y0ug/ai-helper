package gemini

import (
	"github.com/y0ug/ai-helper/pkg/llmclient/http/requestoption"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/openai"
	"github.com/y0ug/ai-helper/pkg/llmclient/types"
)

type GeminiProvider struct {
	*openai.OpenAIProvider
}

func NewGeminiProvider(opts ...requestoption.RequestOption) types.LLMProvider {
	return &GeminiProvider{
		&openai.OpenAIProvider{
			Client: NewClient(opts...).Client,
		},
	}
}
