package anthropic

import (
	"context"
	"encoding/json"

	"github.com/y0ug/ai-helper/pkg/llmclient/http/requestoption"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/streaming"
	"github.com/y0ug/ai-helper/pkg/llmclient/types"
)

type AnthropicProvider struct {
	client *Client
}

func NewAnthropicProvider(opts ...requestoption.RequestOption) types.LLMProvider {
	return &AnthropicProvider{
		client: NewClient(opts...),
	}
}

func AnthropicMessageToChatMessage(am *Message) *types.ChatMessage {
	cm := &types.ChatMessage{}
	cm.ID = am.ID
	cm.Model = am.Model
	cm.Usage = &types.ChatMessageUsage{}
	cm.Usage.InputTokens = am.Usage.InputTokens
	cm.Usage.OutputTokens = am.Usage.OutputTokens

	c := types.ChatMessageChoice{}
	c.Content = append(c.Content, am.Content...)
	c.Role = am.Role
	c.StopReason = am.StopReason
	cm.Choice = append(cm.Choice, c)
	return cm
}

func (a *AnthropicProvider) Send(
	ctx context.Context,
	params types.ChatMessageNewParams,
) (*types.ChatMessage, error) {
	paramsProvider := BaseChatMessageNewParamsToAnthropic(params)
	am, err := a.client.Message.New(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}

	return AnthropicMessageToChatMessage(&am), nil
}

func (a *AnthropicProvider) Stream(
	ctx context.Context,
	params types.ChatMessageNewParams,
) (streaming.Streamer[types.EventStream], error) {
	paramsProvider := BaseChatMessageNewParamsToAnthropic(params)
	stream, err := a.client.Message.NewStreaming(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}
	return types.NewProviderEventStream[MessageStreamEvent](
		stream,
		NewAnthropicEventHandler(),
	), nil
}

func BaseChatMessageNewParamsToAnthropic(
	params types.ChatMessageNewParams,
) MessageNewParams {
	systemPromt := ""
	msgs := make([]MessageParam, 0)
	for _, m := range params.Messages {
		if m.Role == "system" {
			systemPromt = m.Content[0].String()
			continue
		}
		msgs = append(msgs, MessageParam{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	paramsProvider := MessageNewParams{
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
		var delta types.AIContent
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
