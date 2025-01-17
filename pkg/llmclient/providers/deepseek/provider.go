package deepseek

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/requestoption"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/streaming"
	"github.com/y0ug/ai-helper/pkg/llmclient/providers/openai"
)

type DeepseekProvider struct {
	client *Client
}

func NewDeepSeekProvider(opts ...requestoption.RequestOption) common.LLMProvider {
	return &DeepseekProvider{
		client: NewClient(opts...),
	}
}

func (a *DeepseekProvider) Send(
	ctx context.Context,
	params common.ChatMessageNewParams,
) (*common.ChatMessage, error) {
	paramsProvider := openai.BaseChatMessageNewParamsToOpenAI(params)

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
					openai.FromOpenaiToolCallToAIContent(call),
				)
			}

			if choice.Message.Content != "" {
				c.Content = append(c.Content, common.NewTextContent(choice.Message.Content))
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
	params common.ChatMessageNewParams,
) (streaming.Streamer[common.EventStream], error) {
	paramsProvider := openai.BaseChatMessageNewParamsToOpenAI(params)

	stream, err := a.client.Chat.NewStreaming(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}
	return common.NewProviderEventStream(
		stream,
		openai.NewOpenAIEventHandler(),
	), nil
}
