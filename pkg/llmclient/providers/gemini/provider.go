package gemini

import (
	"github.com/y0ug/ai-helper/pkg/llmclient/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/requestoption"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/openai"
)

type GeminiProvider struct {
	*openai.OpenAIProvider
}

func NewGeminiProvider(opts ...requestoption.RequestOption) common.LLMProvider {
	return &GeminiProvider{
		&openai.OpenAIProvider{
			Client: NewClient(opts...).Client,
		},
	}
}
