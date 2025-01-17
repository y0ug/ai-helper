package anthropic

import (
	"encoding/json"

	"github.com/y0ug/ai-helper/pkg/llmclient/types"
)

// AnthropicEventHandler processes Anthropic-specific events
type AnthropicEventHandler struct {
	message Message
}

func NewAnthropicEventHandler() *AnthropicEventHandler {
	return &AnthropicEventHandler{}
}

func (h *AnthropicEventHandler) ShouldContinue(event MessageStreamEvent) bool {
	return true // event.Type != "message_stop"
}

func (h *AnthropicEventHandler) HandleEvent(
	event MessageStreamEvent,
) (types.EventStream, error) {
	h.message.Accumulate(event)
	evt := types.EventStream{Type: event.Type}

	switch event.Type {
	case "content_block_delta":
		var delta types.MessageContent
		if err := json.Unmarshal(event.Delta, &delta); err != nil {
			return evt, nil
		}
		if delta.Type == "text_delta" {
			evt.Type = "text_delta"
			evt.Delta = delta.Text
		}
	case "message_stop":
		evt.Message = AnthropicMessageToChatMessage(&h.message)
	}
	return evt, nil
}
