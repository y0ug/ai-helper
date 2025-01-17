package gemini

import (
	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/openai"
	"github.com/y0ug/ai-helper/pkg/llmclient/types"
)

type Provider struct {
	*openai.Provider
}

func New(opts ...options.RequestOption) types.LLMProvider {
	return &Provider{
		&openai.Provider{
			Client: NewClient(opts...).Client,
		},
	}
}
