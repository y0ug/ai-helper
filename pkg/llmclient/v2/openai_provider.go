package llmclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/y0ug/ai-helper/pkg/llmclient/v2/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/openai"
)

type OpenAIProvider struct {
	client *openai.Client
}

func (a *OpenAIProvider) Send(
	ctx context.Context,
	params common.BaseChatMessageNewParams,
) (*common.BaseChatMessage, error) {
	paramsOpenAI := openai.ChatCompletionNewParams{
		Model:               params.Model,
		MaxCompletionTokens: &params.MaxTokens,
		Temperature:         params.Temperature,
		Messages:            FromLLMMessageToOpenAi(params.Messages...),
	}

	resp, err := a.client.Chat.New(ctx, paramsOpenAI)
	if err != nil {
		return nil, err
	}

	fmt.Printf("%v\n", resp)
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

func FromLLMToolToOpenAI(tool common.LLMTool) openai.Tool {
	var desc *string
	if tool.Description != nil {
		descCopy := *tool.Description
		desc = &descCopy
		if len(*desc) > 512 {
			foo := descCopy[:512]
			desc = &foo
		}
	}
	aiTool := openai.Tool{
		Type: "function",
		Function: openai.ToolFunction{
			Name:        tool.Name,
			Description: desc,
			Parameters:  tool.InputSchema,
		},
	}
	return aiTool
}

func FromOpenaiToolCallToAIContent(t openai.ToolCall) *common.AIContent {
	var args map[string]interface{}
	_ = json.Unmarshal([]byte(t.Function.Arguments), &args)
	return common.NewToolUseContent(t.ID, t.Function.Name, args)
}

func FromLLMMessageToOpenAi(m ...common.BaseChatMessageParams) []openai.ChatCompletionMessageParam {
	userMessages := make([]openai.ChatCompletionMessageParam, 0)
	for _, msg := range m {
		content := msg.Content[0]
		if content.Type == common.ContentTypeToolResult {
			userMessages = append(userMessages, openai.ChatCompletionMessageParam{
				Role:       "tool",
				Content:    content.Content,
				ToolCallID: content.ToolUseID,
			})
		} else {
			userMessages = append(userMessages, openai.ChatCompletionMessageParam{
				Role:    msg.GetRole(),
				Content: content.String(),
			})
		}
	}
	return userMessages
}
