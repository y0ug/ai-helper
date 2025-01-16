package llmclient

import (
	"context"
	"encoding/json"

	"github.com/y0ug/ai-helper/pkg/llmclient/anthropic"
	"github.com/y0ug/ai-helper/pkg/llmclient/common"
)

type AnthropicProvider struct {
	client *anthropic.Client
}

func AnthropicMessageToChatMessage(am *anthropic.Message) *common.BaseChatMessage {
	cm := &common.BaseChatMessage{}
	cm.ID = am.ID
	cm.Model = am.Model
	cm.Usage = &common.BaseChatMessageUsage{}
	cm.Usage.InputTokens = am.Usage.InputTokens
	cm.Usage.OutputTokens = am.Usage.OutputTokens

	c := common.BaseChatMessageChoice{}
	c.Content = append(c.Content, am.Content...)
	c.Role = am.Role
	c.FinishReason = am.StopReason
	cm.Choice = append(cm.Choice, c)
	return cm
}

func (a *AnthropicProvider) Send(
	ctx context.Context,
	params common.BaseChatMessageNewParams,
) (*common.BaseChatMessage, error) {
	paramsProvider := BaseChatMessageNewParamsToAnthropic(params)
	am, err := a.client.Message.New(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}

	return AnthropicMessageToChatMessage(&am), nil
}

func (a *AnthropicProvider) Stream(
	ctx context.Context,
	params common.BaseChatMessageNewParams,
) (common.Streamer[common.StreamEvent], error) {
	paramsProvider := BaseChatMessageNewParamsToAnthropic(params)
	stream, err := a.client.Message.NewStreaming(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}
	return common.NewWrapperStream[anthropic.MessageStreamEvent](
		stream,
		NewAnthropicEventHandler(),
	), nil
}

func BaseChatMessageNewParamsToAnthropic(
	params common.BaseChatMessageNewParams,
) anthropic.MessageNewParams {
	systemPromt := ""
	msgs := make([]anthropic.MessageParam, 0)
	for _, m := range params.Messages {
		if m.Role == "system" {
			systemPromt = m.Content[0].String()
			continue
		}
		msgs = append(msgs, anthropic.MessageParam{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	paramsProvider := anthropic.MessageNewParams{
		Model:       params.Model,
		MaxTokens:   params.MaxTokens,
		Temperature: params.Temperature,
		Messages:    msgs,
		System:      systemPromt,
		Tools:       params.Tools,
	}
	return paramsProvider
}

// AnthropicEventHandler processes Anthropic-specific events
type AnthropicEventHandler struct {
	message anthropic.Message
}

func NewAnthropicEventHandler() *AnthropicEventHandler {
	return &AnthropicEventHandler{}
}

func (h *AnthropicEventHandler) ProcessEvent(data []byte) common.StreamEvent {
	var event anthropic.MessageStreamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return common.StreamEvent{Type: "error", Delta: err}
	}

	h.message.Accumulate(event)
	evt := common.StreamEvent{Type: event.Type}

	switch event.Type {
	case "content_block_delta":
		var delta common.AIContent
		if err := json.Unmarshal(event.Delta, &delta); err != nil {
			return evt
		}
		if delta.Type == "text_delta" {
			evt.Type = "text_delta"
			evt.Delta = delta.Text
		}
	case "message_stop":
		evt.Message = AnthropicMessageToChatMessage(&h.message)
	}
	return evt
}
