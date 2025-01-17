package openai

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/streaming"
	"github.com/y0ug/ai-helper/pkg/llmclient/types"
)

type OpenAIProvider struct {
	Client *Client
}

func NewOpenAIProvider(opts ...options.RequestOption) types.LLMProvider {
	return &OpenAIProvider{
		Client: NewClient(opts...),
	}
}

func (a *OpenAIProvider) Send(
	ctx context.Context,
	params types.ChatParams,
) (*types.ChatResponse, error) {
	paramsProvider := ToChatCompletionNewParams(params)

	resp, err := a.Client.Chat.New(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}

	return ToChatResponse(&resp), nil
}

func (a *OpenAIProvider) Stream(
	ctx context.Context,
	params types.ChatParams,
) (streaming.Streamer[types.EventStream], error) {
	paramsProvider := ToChatCompletionNewParams(params)

	stream, err := a.Client.Chat.NewStreaming(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}
	return types.NewProviderEventStream(
		stream,
		NewOpenAIEventHandler(),
	), nil
}
