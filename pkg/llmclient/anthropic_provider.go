package llmclient

import (
	"context"
	"encoding/json"

	"github.com/y0ug/ai-helper/pkg/llmclient/anthropic"
	"github.com/y0ug/ai-helper/pkg/llmclient/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/stream"
)

type AnthropicProvider struct {
	client *anthropic.Client
}

func AnthropicMessageToChatMessage(am *anthropic.Message) *common.ChatMessage {
	cm := &common.ChatMessage{}
	cm.ID = am.ID
	cm.Model = am.Model
	cm.Usage = &common.ChatMessageUsage{}
	cm.Usage.InputTokens = am.Usage.InputTokens
	cm.Usage.OutputTokens = am.Usage.OutputTokens

	c := common.ChatMessageChoice{}
	c.Content = append(c.Content, am.Content...)
	c.Role = am.Role
	c.StopReason = am.StopReason
	cm.Choice = append(cm.Choice, c)
	return cm
}

func (a *AnthropicProvider) Send(
	ctx context.Context,
	params common.ChatMessageNewParams,
) (*common.ChatMessage, error) {
	paramsProvider := BaseChatMessageNewParamsToAnthropic(params)
	am, err := a.client.Message.New(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}

	return AnthropicMessageToChatMessage(&am), nil
}

func (a *AnthropicProvider) Stream(
	ctx context.Context,
	params common.ChatMessageNewParams,
) (stream.Streamer[common.EventStream], error) {
	paramsProvider := BaseChatMessageNewParamsToAnthropic(params)
	stream, err := a.client.Message.NewStreaming(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}
	return common.NewProviderEventStream[anthropic.MessageStreamEvent](
		stream,
		NewAnthropicEventHandler(),
	), nil
}

func BaseChatMessageNewParamsToAnthropic(
	params common.ChatMessageNewParams,
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

func (h *AnthropicEventHandler) ShouldContinue(event anthropic.MessageStreamEvent) bool {
	return true // event.Type != "message_stop"
}

func (h *AnthropicEventHandler) HandleEvent(
	event anthropic.MessageStreamEvent,
) (common.EventStream, error) {
	h.message.Accumulate(event)
	evt := common.EventStream{Type: event.Type}

	switch event.Type {
	case "content_block_delta":
		var delta common.AIContent
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
