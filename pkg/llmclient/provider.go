package llmclient

import (
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
	"github.com/y0ug/ai-helper/pkg/llmclient/modelinfo"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/anthropic"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/deepseek"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/gemini"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/openai"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/openrouter"
)

// New provider factory
func New(
	modelName string,
	infoProvider modelinfo.Provider,
	requestOpts ...options.RequestOption,
) (chat.Provider, *modelinfo.Model) {
	model, err := modelinfo.Parse(modelName, infoProvider)
	if err != nil {
		return nil, nil
	}

	var provider chat.Provider
	switch model.Provider {
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

	return provider, model
}
