package anthropic

import (
	"encoding/json"
	"fmt"

	"github.com/y0ug/ai-helper/pkg/llmclient/request/ssestream"
)

// AnthropicStreamHandler implements BaseStreamHandler for Anthropic's streaming responses
type AnthropicStreamHandler struct{}

func NewAnthropicStreamHandler() *AnthropicStreamHandler {
	return &AnthropicStreamHandler{}
}

func (h *AnthropicStreamHandler) HandleEvent(event ssestream.Event) (MessageStreamEvent, error) {
	var result MessageStreamEvent

	switch event.Type {
	case "completion":
		if err := json.Unmarshal(event.Data, &result); err != nil {
			return result, err
		}
	case "message_start",
		"message_delta",
		"message_stop",
		"content_block_start",
		"content_block_delta",
		"content_block_stop":
		if err := json.Unmarshal(event.Data, &result); err != nil {
			return result, err
		}
	case "error":
		return result, fmt.Errorf("received error while streaming: %s", string(event.Data))
	}

	return result, nil
}

func (h *AnthropicStreamHandler) ShouldContinue(event ssestream.Event) bool {
	if event.Type == "ping" {
		return true
	}
	return event.Type != "error"
}
