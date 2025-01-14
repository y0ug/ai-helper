package llmclient

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/v2/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/deepseek"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/ssestream"
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
	params common.BaseChatMessageNewParams,
) (*common.BaseChatMessage, error) {
	paramsProvider := BaseChatMessageNewParamsToOpenAI(params)

	resp, err := a.client.Chat.New(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}

	ret := &common.BaseChatMessage{}
	ret.ID = resp.ID
	ret.Model = resp.Model
	ret.Usage = &common.BaseChatMessageUsage{}
	ret.Usage.InputTokens = resp.Usage.PromptTokens
	ret.Usage.OutputTokens = resp.Usage.CompletionTokens
	if len(resp.Choices) > 0 {
		for _, choice := range resp.Choices {
			c := common.BaseChatMessageChoice{}
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
			c.FinishReason = choice.FinishReason

			ret.Choice = append(ret.Choice, c)
		}
	}
	return ret, nil
}

func (a *DeepseekProvider) Stream(
	ctx context.Context,
	params common.BaseChatMessageNewParams,
) (ssestream.Streamer[common.LLMStreamEvent], error) {
	paramsProvider := BaseChatMessageNewParamsToOpenAI(params)

	_ = a.client.Chat.NewStreaming(ctx, paramsProvider)

	return nil, nil
}
