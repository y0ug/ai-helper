package llmclient

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/common"
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
