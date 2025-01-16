package llmclient

import (
	"context"
	"encoding/json"

	"github.com/y0ug/ai-helper/pkg/llmclient/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/openai"
)

type OpenAIProvider struct {
	client *openai.Client
}

func OpenaiChatCompletionToChatMessage(cc *openai.ChatCompletion) *common.BaseChatMessage {
	cm := &common.BaseChatMessage{}
	cm.ID = cc.ID
	cm.Model = cc.Model
	cm.Usage = &common.BaseChatMessageUsage{}
	cm.Usage.InputTokens = cc.Usage.PromptTokens
	cm.Usage.OutputTokens = cc.Usage.CompletionTokens
	for _, choice := range cc.Choices {
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

		cm.Choice = append(cm.Choice, c)
	}
	return cm
}

func BaseChatMessageNewParamsToOpenAI(
	params common.BaseChatMessageNewParams,
) openai.ChatCompletionNewParams {
	return openai.ChatCompletionNewParams{
		Model:               params.Model,
		MaxCompletionTokens: &params.MaxTokens,
		Temperature:         params.Temperature,
		N:                   params.N,
		Messages:            FromLLMMessageToOpenAi(params.Messages...),
		Tools:               FromLLMToolToOpenAI(params.Tools...),
	}
}

func (a *OpenAIProvider) Send(
	ctx context.Context,
	params common.BaseChatMessageNewParams,
) (*common.BaseChatMessage, error) {
	paramsProvider := BaseChatMessageNewParamsToOpenAI(params)

	resp, err := a.client.Chat.New(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}

	return OpenaiChatCompletionToChatMessage(&resp), nil
}

func (a *OpenAIProvider) Stream(
	ctx context.Context,
	params common.BaseChatMessageNewParams,
) common.Streamer[common.LLMStreamEvent] {
	paramsProvider := BaseChatMessageNewParamsToOpenAI(params)

	stream := a.client.Chat.NewStreaming(ctx, paramsProvider)
	return common.NewWrapperStream[openai.ChatCompletionChunk](stream, "openai")
}

func FromLLMToolToOpenAI(tools ...common.Tool) []openai.Tool {
	result := make([]openai.Tool, 0)
	for _, tool := range tools {
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
		result = append(result, aiTool)

	}
	return result
}

func FromOpenaiToolCallToAIContent(t openai.ToolCall) *common.AIContent {
	// var args map[string]interface{}
	// _ = json.Unmarshal([]byte(t.Function.Arguments), &args)
	return common.NewToolUseContent(t.ID, t.Function.Name, json.RawMessage(t.Function.Arguments))
}

func AIContentToolCallsToOpenAI(t ...*common.AIContent) []openai.ToolCall {
	d := make([]openai.ToolCall, 0)
	for _, content := range t {
		if content.Type == common.ContentTypeToolUse {
			d = append(d, openai.ToolCall{
				ID:   content.ID,
				Type: "function",
				Function: openai.FunctionCall{
					Name:      content.Name,
					Arguments: string(content.Input),
				},
			})
		}
	}
	return d
}

func FromLLMMessageToOpenAi(
	m ...*common.BaseChatMessageParams,
) []openai.ChatCompletionMessageParam {
	userMessages := make([]openai.ChatCompletionMessageParam, 0)
	for _, msg := range m {
		content := msg.Content[0]
		if content.Type == common.ContentTypeToolUse {
			// For toolCalls we need to process all of them in one time
			userMessages = append(userMessages, openai.ChatCompletionMessageParam{
				Role:      "assistant",
				ToolCalls: AIContentToolCallsToOpenAI(msg.Content...),
			})
		} else if content.Type == common.ContentTypeToolResult {
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
