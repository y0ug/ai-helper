package types

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/http/streaming"
)

type ChatMessageNewParams struct {
	Model       string
	MaxTokens   int
	Temperature float64
	Messages    []*ChatMessageParams
	Stream      bool
	Tools       []Tool
	ToolChoice  string // auto, any, tool
	N           *int   // number of choice
}

type ChatMessage struct {
	ID     string              `json:"id,omitempty"`
	Choice []ChatMessageChoice `json:"choice,omitempty"`
	Usage  *ChatMessageUsage   `json:"usage,omitempty"`
	Model  string              `json:"model,omitempty"`
}

func (cm *ChatMessage) ToMessageParams() *ChatMessageParams {
	return &ChatMessageParams{
		Content: cm.Choice[0].Content,
		Role:    cm.Choice[0].Role,
	}
}

// StopReason is the reason the model stopped generating messages. It can be one of:
// - `"end_turn"`: the model reached a natural stopping point
// - `"max_tokens"`: we exceeded the requested `max_tokens` or the model's maximum
// - `"stop_sequence"`: one of your provided custom `stop_sequences` was generated
// - `"tool_use"`: the model invoked one or more tools
type ChatMessageChoice struct {
	Role       string       `json:"role,omitempty"` // Always "assistant"
	Content    []*AIContent `json:"content,omitempty"`
	StopReason string       `json:"stop_reason,omitempty"`
}

type ChatMessageUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type ChatMessageParams struct {
	Role    string       `json:"role"`
	Content []*AIContent `json:"content"`
}

func (m ChatMessageParams) GetRole() string {
	return m.Role
}

func (m ChatMessageParams) GetContents() []*AIContent {
	return m.Content
}

func (m ChatMessageParams) GetContent() *AIContent {
	if len(m.Content) != 0 {
		return m.Content[0]
	}
	return nil
}

type Tool struct {
	Description *string     `json:"description,omitempty"`
	InputSchema interface{} `json:"input_schema,omitempty"`
	Name        string      `json:"name"`
}

// WithModel sets the model for BaseChatMessageNewParams
func WithModel(model string) func(*ChatMessageNewParams) {
	return func(p *ChatMessageNewParams) {
		p.Model = model
	}
}

// WithMaxTokens sets the max tokens for BaseChatMessageNewParams
func WithMaxTokens(tokens int) func(*ChatMessageNewParams) {
	return func(p *ChatMessageNewParams) {
		p.MaxTokens = tokens
	}
}

// WithTemperature sets the temperature for BaseChatMessageNewParams
func WithTemperature(temp float64) func(*ChatMessageNewParams) {
	return func(p *ChatMessageNewParams) {
		p.Temperature = temp
	}
}

// WithMessages sets the messages for BaseChatMessageNewParams
func WithMessages(
	messages ...*ChatMessageParams,
) func(*ChatMessageNewParams) {
	return func(p *ChatMessageNewParams) {
		p.Messages = messages
	}
}

// WithTools sets the tools/functions for BaseChatMessageNewParams
func WithTools(tools ...Tool) func(*ChatMessageNewParams) {
	return func(p *ChatMessageNewParams) {
		p.Tools = tools
	}
}

func NewMessageParams(role string, content ...*AIContent) *ChatMessageParams {
	return &ChatMessageParams{
		Role:    role,
		Content: content,
	}
}

func NewSystemMessageParams(text string) *ChatMessageParams {
	return NewMessageParams("system", NewTextContent(text))
}

func NewUserMessageParams(text string) *ChatMessageParams {
	return NewMessageParams("user", NewTextContent(text))
}

// NewChatMessageParams creates a new BaseChatMessageNewParams with the given options
func NewChatMessageParams(
	opts ...func(*ChatMessageNewParams),
) *ChatMessageNewParams {
	params := &ChatMessageNewParams{}
	for _, opt := range opts {
		opt(params)
	}
	return params
}

func StreamChatMessageToChannel(
	ctx context.Context,
	stream streaming.Streamer[EventStream],
	ch chan<- EventStream,
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
