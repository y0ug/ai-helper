package llmclient

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/deepseek"
	"github.com/y0ug/ai-helper/pkg/llmclient/openai"
	"github.com/y0ug/ai-helper/pkg/llmclient/stream"
)

type OpenRouterProvider struct {
	*OpenAIProvider
}

type GeminiProvider struct {
	*OpenAIProvider
}

type DeepseekProvider struct {
	client *deepseek.Client
}

func (a *DeepseekProvider) Send(
	ctx context.Context,
	params common.ChatMessageNewParams,
) (*common.ChatMessage, error) {
	paramsProvider := BaseChatMessageNewParamsToOpenAI(params)

	resp, err := a.client.Chat.New(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}

	ret := &common.ChatMessage{}
	ret.ID = resp.ID
	ret.Model = resp.Model
	ret.Usage = &common.ChatMessageUsage{}
	ret.Usage.InputTokens = resp.Usage.PromptTokens
	ret.Usage.OutputTokens = resp.Usage.CompletionTokens
	if len(resp.Choices) > 0 {
		for _, choice := range resp.Choices {
			c := common.ChatMessageChoice{}
			for _, call := range choice.Message.ToolCalls {
				c.Content = append(
					c.Content,
					FromOpenaiToolCallToAIContent(call),
				)
			}

			if choice.Message.Content != "" {
				c.Content = append(c.Content, common.NewTextContent(choice.Message.Content))
			}

			// Role is not choice is our model
			c.Role = choice.Message.Role
			c.StopReason = OpenaAIFinishReasonToStopReason(choice.FinishReason)

			ret.Choice = append(ret.Choice, c)
		}
	}
	return ret, nil
}

func (a *DeepseekProvider) Stream(
	ctx context.Context,
	params common.ChatMessageNewParams,
) (stream.Streamer[common.EventStream], error) {
	paramsProvider := BaseChatMessageNewParamsToOpenAI(params)

	stream, err := a.client.Chat.NewStreaming(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}
	return common.NewProviderEventStream[openai.ChatCompletionChunk](
		stream,
		NewOpenAIEventHandler(),
	), nil
}
