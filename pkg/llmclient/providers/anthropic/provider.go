package anthropic

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/streaming"
	"github.com/y0ug/ai-helper/pkg/llmclient/types"
)

type Provider struct {
	client *Client
}

func New(opts ...options.RequestOption) types.LLMProvider {
	return &Provider{
		client: NewClient(opts...),
	}
}

func (a *Provider) Send(
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

func (a *Provider) Stream(
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
