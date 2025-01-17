package llmclient

import (
	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/anthropic"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/deepseek"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/gemini"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/openai"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/openrouter"
	"github.com/y0ug/ai-helper/pkg/llmclient/types"
)

func NewProviderByModel(
	modelName string,
	infoProvider ModelInfoProvider,
	requestOpts ...options.RequestOption,
) (types.LLMProvider, *Model) {
	model, err := ParseModel(modelName, infoProvider)
	if err != nil {
		return nil, nil
	}

	var provider types.LLMProvider
	switch model.Provider {
	case "anthropic":
		provider = anthropic.NewAnthropicProvider(requestOpts...)
	case "openrouter":
		provider = openrouter.NewOpenRouterProvider(requestOpts...)
	case "openai":
		provider = openai.NewOpenAIProvider(requestOpts...)
	case "gemini":
		provider = gemini.NewGeminiProvider(requestOpts...)
	case "deepseek":
		provider = deepseek.NewDeepSeekProvider(requestOpts...)
	}

	return provider, model
}
