package anthropic

import (
	"context"
	"net/http"

	"github.com/y0ug/ai-helper/pkg/llmclient/v2/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestconfig"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestoption"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/ssestream"
)

type MessageParam struct {
	Role    string              `json:"role"`
	Content []*common.AIContent `json:"content"`
}

// Message response, ToParam methode convert to MessageParam
type Message struct {
	ID           string              `json:"id,omitempty"`
	Content      []*common.AIContent `json:"content,omitempty"`
	Role         string              `json:"role,omitempty"` // Always "assistant"
	StopReason   string              `json:"stop_reason,omitempty"`
	StopSequence string              `json:"stop_sequence,omitempty"`
	Type         string              `json:"type,omitempty"` // Always "message"
	Usage        *Usage              `json:"usage,omitempty"`
	Model        string              `json:"model,omitempty"`
}

func (r *Message) ToParam() MessageParam {
	return MessageParam{
		Role:    r.Role,
		Content: r.Content,
	}
}

type MessageStreamEvent struct {
	Type string `json:"type"`
	// This field can have the runtime type of [ContentBlockStartEventContentBlock].
	ContentBlock map[string]string `json:"content_block"`
	// This field can have the runtime type of [MessageDeltaEventDelta],
	// [ContentBlockDeltaEventDelta].
	Delta   *common.AIContent `json:"delta"`
	Index   int64             `json:"index"`
	Message Message           `json:"message"`
	// Billing and rate-limit usage.
	//
	// Anthropic's API bills and rate-limits by token counts, as tokens represent the
	// underlying cost to our systems.
	//
	// Under the hood, the API transforms requests into a format suitable for the
	// model. The model's output then goes through a parsing stage before becoming an
	// API response. As a result, the token counts in `usage` will not match one-to-one
	// with the exact visible content of an API request or response.
	//
	// For example, `output_tokens` will be non-zero, even for an empty string response
	// from Claude.
	// Usage MessageDeltaUsage `json:"usage"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type MessageNewParams struct {
	MaxTokens     int            `json:"max_tokens,omitempty"`
	Messages      []MessageParam `json:"messages"` // MessageParam
	Model         string         `json:"model"`
	StopSequences []string       `json:"stop_sequences,omitempty"`
	Stream        bool           `json:"stream,omitempty"`
	System        string         `json:"system,omitempty"`

	Temperature float64          `json:"temperature,omitempty"` // Number between 0 and 1 that controls randomness of the output.
	Tools       []common.LLMTool `json:"tools,omitempty"`       // ToolParam
	ToolChoice  interface{}      `json:"tool_choice,omitempty"` // Auto but can be used to force to used a tools
}

// MessageService contains methods and other services that help with interacting
// with the anthropic API.
//
// Note, unlike clients, this service does not read variables from the environment
// automatically. You should not instantiate this service directly, and instead use
// the [NewMessageService] method instead.
type MessageService struct {
	Options []requestoption.RequestOption
}

// NewMessageService generates a new service that applies the given options to each
// request. These options are applied after the parent client's options (if there
// is one), and before any request-specific options.
func NewMessageService(opts ...requestoption.RequestOption) (r *MessageService) {
	r = &MessageService{}
	r.Options = opts
	return
}

// Send a structured list of input messages with text and/or image content, and the
// model will generate the next message in the conversation.
//
// The Messages API can be used for either single queries or stateless multi-turn
// conversations.
//
// Note: If you choose to set a timeout for this request, we recommend 10 minutes.
func (r *MessageService) New(
	ctx context.Context,
	body MessageNewParams,
	opts ...requestoption.RequestOption,
) (res *Message, err error) {
	opts = append(r.Options[:], opts...)
	path := "v1/messages"
	err = requestconfig.ExecuteNewRequest(
		ctx,
		http.MethodPost,
		path,
		body,
		&res,
		NewAPIErrorAnthropic,
		opts...)
	return
}

// Send a structured list of input messages with text and/or image content, and the
// model will generate the next message in the conversation.
//
// The Messages API can be used for either single queries or stateless multi-turn
// conversations.
//
// Note: If you choose to set a timeout for this request, we recommend 10 minutes.
func (r *MessageService) NewStreaming(
	ctx context.Context,
	body MessageNewParams,
	opts ...requestoption.RequestOption,
) ssestream.Streamer[MessageStreamEvent] {
	var (
		raw *http.Response
		err error
	)
	opts = append(r.Options[:], opts...)
	opts = append([]requestoption.RequestOption{requestoption.WithJSONSet("stream", true)}, opts...)
	path := "v1/messages"
	err = requestconfig.ExecuteNewRequest(
		ctx,
		http.MethodPost,
		path,
		body,
		&raw,
		NewAPIErrorAnthropic,
		opts...)
	return ssestream.NewAnthropicStream[MessageStreamEvent](ssestream.NewDecoder(raw), err)
}
