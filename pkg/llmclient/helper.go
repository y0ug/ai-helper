package llmclient

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/anthropic"
	"github.com/y0ug/ai-helper/pkg/llmclient/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/deepseek"
	"github.com/y0ug/ai-helper/pkg/llmclient/gemini"
	"github.com/y0ug/ai-helper/pkg/llmclient/openai"
	"github.com/y0ug/ai-helper/pkg/llmclient/requestoption"
	"github.com/y0ug/ai-helper/pkg/llmclient/stream"
)

// WithModel sets the model for BaseChatMessageNewParams
func WithModel(model string) func(*common.ChatMessageNewParams) {
	return func(p *common.ChatMessageNewParams) {
		p.Model = model
	}
}

// WithMaxTokens sets the max tokens for BaseChatMessageNewParams
func WithMaxTokens(tokens int) func(*common.ChatMessageNewParams) {
	return func(p *common.ChatMessageNewParams) {
		p.MaxTokens = tokens
	}
}

// WithTemperature sets the temperature for BaseChatMessageNewParams
func WithTemperature(temp float64) func(*common.ChatMessageNewParams) {
	return func(p *common.ChatMessageNewParams) {
		p.Temperature = temp
	}
}

// WithMessages sets the messages for BaseChatMessageNewParams
func WithMessages(
	messages ...*common.ChatMessageParams,
) func(*common.ChatMessageNewParams) {
	return func(p *common.ChatMessageNewParams) {
		p.Messages = messages
	}
}

// WithTools sets the tools/functions for BaseChatMessageNewParams
func WithTools(tools ...common.Tool) func(*common.ChatMessageNewParams) {
	return func(p *common.ChatMessageNewParams) {
		p.Tools = tools
	}
}

// NewUserMessage creates a new user message
func NewUserMessage(text string) *common.ChatMessageParams {
	return &common.ChatMessageParams{
		Role: "user",
		Content: []*common.AIContent{
			common.NewTextContent(
				text,
			),
		},
	}
}

func NewUserMessageContent(content ...*common.AIContent) *common.ChatMessageParams {
	return &common.ChatMessageParams{
		Role:    "user",
		Content: content,
	}
}

// NewSystemMessage creates a new system message
func NewSystemMessage(text string) *common.ChatMessageParams {
	return &common.ChatMessageParams{
		Role: "system",
		Content: []*common.AIContent{
			common.NewTextContent(
				text,
			),
		},
	}
}

// NewMessagesParams creates a new slice of message parameters
func NewMessagesParams(msg ...*common.ChatMessageParams) []*common.ChatMessageParams {
	return msg
}

// NewChatParams creates a new BaseChatMessageNewParams with the given options
func NewChatParams(
	opts ...func(*common.ChatMessageNewParams),
) *common.ChatMessageNewParams {
	params := &common.ChatMessageNewParams{}
	for _, opt := range opts {
		opt(params)
	}
	return params
}

func NewDeepSeekProvider(opts ...requestoption.RequestOption) common.LLMProvider {
	return &DeepseekProvider{
		client: deepseek.NewClient(opts...),
	}
}

func NewAnthropicProvider(opts ...requestoption.RequestOption) common.LLMProvider {
	return &AnthropicProvider{
		client: anthropic.NewClient(opts...),
	}
}

func NewOpenAIProvider(opts ...requestoption.RequestOption) common.LLMProvider {
	return &OpenAIProvider{
		client: openai.NewClient(opts...),
	}
}

func NewOpenRouterProvider(opts ...requestoption.RequestOption) common.LLMProvider {
	return &GeminiProvider{
		&OpenAIProvider{
			client: gemini.NewClient(opts...).Client,
		},
	}
}

func NewGeminiProvider(opts ...requestoption.RequestOption) common.LLMProvider {
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
) (common.LLMProvider, *Model) {
	model, err := ParseModel(modelName, infoProvider)
	if err != nil {
		return nil, nil
	}

	switch model.Provider {
	case "anthropic":
		return NewAnthropicProvider(requestOpts...), model
	case "openrouter":
		return NewOpenRouterProvider(requestOpts...), model
	case "openai":
		return NewOpenAIProvider(requestOpts...), model
	case "gemini":
		return NewGeminiProvider(requestOpts...), model
	case "deepseek":
		return NewDeepSeekProvider(requestOpts...), model
	default:
		return nil, model
	}
}

func ConsumeStream(
	ctx context.Context,
	stream stream.Streamer[common.EventStream],
	ch chan<- common.EventStream,
) error {
	defer close(ch)

	for stream.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			ch <- stream.Current()
		}
	}

	return stream.Err()
}
