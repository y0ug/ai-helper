package openrouter

import (
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
	"github.com/y0ug/ai-helper/pkg/llmhaven/http/options"
	"github.com/y0ug/ai-helper/pkg/llmhaven/providers/openai"
)

type Provider struct {
	*openai.Provider
}

func New(opts ...options.RequestOption) chat.Provider {
	return &Provider{
		&openai.Provider{
			Client: NewClient(opts...).Client,
		},
	}
}
