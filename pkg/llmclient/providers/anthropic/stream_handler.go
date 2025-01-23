package anthropic

import (
	"encoding/json"
	"fmt"

	"github.com/y0ug/ai-helper/pkg/llmclient/http/streaming"
)

// AnthropicStreamHandler implements BaseStreamHandler for Anthropic's streaming responses
type AnthropicStreamHandler struct{}

func NewAnthropicStreamHandler() *AnthropicStreamHandler {
	return &AnthropicStreamHandler{}
}

func (h *AnthropicStreamHandler) HandleEvent(event streaming.Event) (MessageStreamEvent, error) {
	var result MessageStreamEvent
	var err error
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
		err = fmt.Errorf("received error while streaming: %s", string(event.Data))
		if err := json.Unmarshal(event.Data, &result); err != nil {
			return result, err
		}
	}

	return result, err
}

func (h *AnthropicStreamHandler) ShouldContinue(event streaming.Event) bool {
	return true
	// if event.Type == "ping" {
	// 	return true
	// }
	// return event.Type != "error"
}
