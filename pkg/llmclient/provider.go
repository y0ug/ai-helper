package llmclient

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/anthropic"
	"github.com/y0ug/ai-helper/pkg/llmclient/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/deepseek"
	"github.com/y0ug/ai-helper/pkg/llmclient/gemini"
	"github.com/y0ug/ai-helper/pkg/llmclient/openai"
	"github.com/y0ug/ai-helper/pkg/llmclient/request/requestoption"
	"github.com/y0ug/ai-helper/pkg/llmclient/stream"
)

type LLMProvider interface {
	// For a single-turn request
	Send(ctx context.Context, params common.ChatMessageNewParams) (*common.ChatMessage, error)

	// For streaming support
	Stream(
		ctx context.Context,
		params common.ChatMessageNewParams,
	) (stream.Streamer[common.EventStream], error)
}

func NewDeepSeekProvider(opts ...requestoption.RequestOption) LLMProvider {
	return &DeepseekProvider{
		client: deepseek.NewClient(opts...),
	}
}

func NewAnthropicProvider(opts ...requestoption.RequestOption) LLMProvider {
	return &AnthropicProvider{
		client: anthropic.NewClient(opts...),
	}
}

func NewOpenAIProvider(opts ...requestoption.RequestOption) LLMProvider {
	return &OpenAIProvider{
		client: openai.NewClient(opts...),
	}
}

func NewOpenRouterProvider(opts ...requestoption.RequestOption) LLMProvider {
	return &GeminiProvider{
		&OpenAIProvider{
			client: gemini.NewClient(opts...).Client,
		},
	}
}

func NewGeminiProvider(opts ...requestoption.RequestOption) LLMProvider {
	return &GeminiProvider{
		&OpenAIProvider{
			client: gemini.NewClient().Client,
		},
	}
}

func NewProviderByModel(
	modelName string,
	infoProvider ModelInfoProvider,
	requestOpts ...requestoption.RequestOption,
) (LLMProvider, *Model) {
	model, err := ParseModel(modelName, infoProvider)
	if err != nil {
		return nil, nil
	}

	var provider LLMProvider
	switch model.Provider {
	case "anthropic":
		provider = NewAnthropicProvider(requestOpts...)
	case "openrouter":
		provider = NewOpenRouterProvider(requestOpts...)
	case "openai":
		provider = NewOpenAIProvider(requestOpts...)
	case "gemini":
		provider = NewGeminiProvider(requestOpts...)
	case "deepseek":
		provider = NewDeepSeekProvider(requestOpts...)
	}

	return provider, model
}
