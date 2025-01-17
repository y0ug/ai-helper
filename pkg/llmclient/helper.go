package llmclient

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/http/streaming"
	"github.com/y0ug/ai-helper/pkg/llmclient/types"
)

// WithModel sets the model for BaseChatMessageNewParams
func WithModel(model string) func(*types.ChatMessageNewParams) {
	return func(p *types.ChatMessageNewParams) {
		p.Model = model
	}
}

// WithMaxTokens sets the max tokens for BaseChatMessageNewParams
func WithMaxTokens(tokens int) func(*types.ChatMessageNewParams) {
	return func(p *types.ChatMessageNewParams) {
		p.MaxTokens = tokens
	}
}

// WithTemperature sets the temperature for BaseChatMessageNewParams
func WithTemperature(temp float64) func(*types.ChatMessageNewParams) {
	return func(p *types.ChatMessageNewParams) {
		p.Temperature = temp
	}
}

// WithMessages sets the messages for BaseChatMessageNewParams
func WithMessages(
	messages ...*types.ChatMessageParams,
) func(*types.ChatMessageNewParams) {
	return func(p *types.ChatMessageNewParams) {
		p.Messages = messages
	}
}

// WithTools sets the tools/functions for BaseChatMessageNewParams
func WithTools(tools ...types.Tool) func(*types.ChatMessageNewParams) {
	return func(p *types.ChatMessageNewParams) {
		p.Tools = tools
	}
}

// NewUserMessage creates a new user message
func NewUserMessage(text string) *types.ChatMessageParams {
	return &types.ChatMessageParams{
		Role: "user",
		Content: []*types.AIContent{
			types.NewTextContent(
				text,
			),
		},
	}
}

func NewUserMessageContent(content ...*types.AIContent) *types.ChatMessageParams {
	return &types.ChatMessageParams{
		Role:    "user",
		Content: content,
	}
}

// NewSystemMessage creates a new system message
func NewSystemMessage(text string) *types.ChatMessageParams {
	return &types.ChatMessageParams{
		Role: "system",
		Content: []*types.AIContent{
			types.NewTextContent(
				text,
			),
		},
	}
}

// NewMessagesParams creates a new slice of message parameters
func NewMessagesParams(msg ...*types.ChatMessageParams) []*types.ChatMessageParams {
	return msg
}

// NewChatParams creates a new BaseChatMessageNewParams with the given options
func NewChatParams(
	opts ...func(*types.ChatMessageNewParams),
) *types.ChatMessageNewParams {
	params := &types.ChatMessageNewParams{}
	for _, opt := range opts {
		opt(params)
	}
	return params
}

func ConsumeStream(
	ctx context.Context,
	stream streaming.Streamer[types.EventStream],
	ch chan<- types.EventStream,
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
