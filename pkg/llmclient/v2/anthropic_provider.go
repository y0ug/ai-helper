package llmclient

import (
	"context"
	"fmt"

	"github.com/y0ug/ai-helper/pkg/llmclient/v2/anthropic"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/common"
)

type AnthropicProvider struct {
	client *anthropic.Client
}

func (a *AnthropicProvider) Send(
	ctx context.Context,
	params common.BaseChatMessageNewParams,
) (*common.BaseChatMessage, error) {
	paramsProvider := BaseChatMessageNewParamsToAnthropic(params)
	resp, err := a.client.Message.New(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}

	fmt.Printf("%v\n", resp)
	ret := &common.BaseChatMessage{}
	ret.ID = resp.ID
	ret.Model = resp.Model
	ret.Usage = &common.BaseChatMessageUsage{}
	ret.Usage.InputTokens = resp.Usage.InputTokens
	ret.Usage.OutputTokens = resp.Usage.OutputTokens

	c := common.BaseChatMessageChoice{}
	c.Content = append(c.Content, resp.Content...)
	c.Role = resp.Role
	c.FinishReason = resp.StopReason
	ret.Choice = append(ret.Choice, c)
	return ret, nil
}

func (a *AnthropicProvider) Stream(
	ctx context.Context,
	params common.BaseChatMessageNewParams,
) common.Streamer[common.LLMStreamEvent] {
	paramsProvider := BaseChatMessageNewParamsToAnthropic(params)
	stream := a.client.Message.NewStreaming(ctx, paramsProvider)

	return common.NewWrapperStream[anthropic.MessageStreamEvent](stream, "anthropic")
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
	}
	return paramsProvider
}
