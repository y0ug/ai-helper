package llmclient

import (
	"context"
	"fmt"

	"github.com/y0ug/ai-helper/pkg/llmclient/v2/anthropic"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/common"
)

type AntropicAdapter struct {
	client *anthropic.Client
}

func (a *AntropicAdapter) Send(
	ctx context.Context,
	params common.BaseChatMessageNewParams,
) (*common.BaseChatMessage, error) {
	systemPromt := ""
	msgs := make([]anthropic.MessageParam, 0)
	for _, m := range params.Messages {
		if m.Role == "system" {
			systemPromt = m.Content[0].String()
		}
		msgs = append(msgs, anthropic.MessageParam{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	paramsProvider := anthropic.MessageNewParams{
		Model:       params.Model,
		MaxTokens:   params.MaxTokens,
		Temperature: int(params.Temperature),
		Messages:    msgs,
		System:      systemPromt,
	}

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
	for _, newContent := range resp.Content {
		c.Content = append(c.Content, newContent)
	}

	// Role is not choice is our model
	c.Role = resp.Role
	c.FinishReason = resp.StopReason
	ret.Choice = append(ret.Choice, c)
	return ret, nil
}
