package llmclient

import (
	"fmt"

	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/anthropic"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/deepseek"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/gemini"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/openai"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/openrouter"
)

// New provider factory
func New(providerName string, requestOpts ...options.RequestOption,
) (chat.Provider, error) {
	var provider chat.Provider
	switch providerName {
	case "anthropic":
		provider = anthropic.New(requestOpts...)
	case "openrouter":
		provider = openrouter.New(requestOpts...)
	case "openai":
		provider = openai.New(requestOpts...)
	case "gemini":
		provider = gemini.New(requestOpts...)
	case "deepseek":
		provider = deepseek.New(requestOpts...)
	}

	if provider == nil {
		return nil, fmt.Errorf("provider %s not found", providerName)
	}
	return provider, nil
}
