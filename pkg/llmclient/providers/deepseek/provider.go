package deepseek

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/streaming"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/openai"
	"github.com/y0ug/ai-helper/pkg/llmclient/types"
)

type DeepseekProvider struct {
	client *Client
}

func NewDeepSeekProvider(opts ...options.RequestOption) types.LLMProvider {
	return &DeepseekProvider{
		client: NewClient(opts...),
	}
}

func (a *DeepseekProvider) Send(
	ctx context.Context,
	params types.ChatMessageNewParams,
) (*types.ChatMessage, error) {
	paramsProvider := openai.BaseChatMessageNewParamsToOpenAI(params)

	resp, err := a.client.Chat.New(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}

	ret := &types.ChatMessage{}
	ret.ID = resp.ID
	ret.Model = resp.Model
	ret.Usage = &types.ChatMessageUsage{}
	ret.Usage.InputTokens = resp.Usage.PromptTokens
	ret.Usage.OutputTokens = resp.Usage.CompletionTokens
	if len(resp.Choices) > 0 {
		for _, choice := range resp.Choices {
			c := types.ChatMessageChoice{}
			for _, call := range choice.Message.ToolCalls {
				c.Content = append(
					c.Content,
					openai.FromOpenaiToolCallToAIContent(call),
				)
			}

			if choice.Message.Content != "" {
				c.Content = append(c.Content, types.NewTextContent(choice.Message.Content))
			}

			// Role is not choice is our model
			c.Role = choice.Message.Role
			c.StopReason = openai.OpenaAIFinishReasonToStopReason(choice.FinishReason)

			ret.Choice = append(ret.Choice, c)
		}
	}
	return ret, nil
}

func (a *DeepseekProvider) Stream(
	ctx context.Context,
	params types.ChatMessageNewParams,
) (streaming.Streamer[types.EventStream], error) {
	paramsProvider := openai.BaseChatMessageNewParamsToOpenAI(params)

	stream, err := a.client.Chat.NewStreaming(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}
	return types.NewProviderEventStream(
		stream,
		openai.NewOpenAIEventHandler(),
	), nil
}
