package anthropic

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/streaming"
	"github.com/y0ug/ai-helper/pkg/llmclient/types"
)

type AnthropicProvider struct {
	client *Client
}

func NewAnthropicProvider(opts ...options.RequestOption) types.LLMProvider {
	return &AnthropicProvider{
		client: NewClient(opts...),
	}
}

func (a *AnthropicProvider) Send(
	ctx context.Context,
	params types.ChatParams,
) (*types.ChatResponse, error) {
	paramsProvider := BaseChatMessageNewParamsToAnthropic(params)
	am, err := a.client.Message.New(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}

	return AnthropicMessageToChatMessage(&am), nil
}

func (a *AnthropicProvider) Stream(
	ctx context.Context,
	params types.ChatParams,
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
