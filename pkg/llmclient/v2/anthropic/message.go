package anthropic

import (
	"encoding/json"
	"fmt"

	"github.com/y0ug/ai-helper/pkg/llmclient/v2/base"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestoption"
)

// ChatCompletionService implements llmclient.ChatService using OpenAI's types.
type MessageService struct {
	*base.BaseChatService[MessageNewParams, Message, MessageStreamEvent]
}

func NewMessageService(opts ...requestoption.RequestOption) *MessageService {
	baseService := &base.BaseChatService[MessageNewParams, Message, MessageStreamEvent]{
		Options:  opts,
		NewError: NewAPIErrorAnthropic,
		Endpoint: "v1/messages",
	}

	return &MessageService{
		BaseChatService: baseService,
	}
}

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

func (a *Message) Accumulate(event MessageStreamEvent) error {
	if a == nil {
		*a = Message{}
	}

	switch event.Type {
	case "message_start":
		*a = event.Message
	case "content_block_start":
		a.Content = append(a.Content, &common.AIContent{})
		data, err := json.Marshal(event.ContentBlock)
		if err != nil {
			return err
		}
		err = json.Unmarshal(data, a.Content[len(a.Content)-1])
		if err != nil {
			return err
		}
	case "content_block_delta":
		if len(a.Content) == 0 {
			return fmt.Errorf(
				"received event of type %s but there was no content block",
				event.Type,
			)
		}
		if event.ContentBlock["type"] == "text" {
			a.Content[len(a.Content)-1].Text += event.Delta.Text
		} else if event.ContentBlock["type"] == "input_json" {
			fmt.Printf("InputJSON: %v\n", event.Delta)
			// a.Content[len(a.Content)-1].ToolUse += event.Delta.ToolUse
		}
	case "message_delta_event":
		fmt.Printf("MessageDelta: %v\n", event.Delta)
	//  update StopRead, StopSequence, Usage
	// a.StopReason = event.Delta.StopReason

	case "content_block_stop":
		if len(a.Content) == 0 {
			return fmt.Errorf(
				"content block finish but final content is empty",
			)
		}
	}

	return nil
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
