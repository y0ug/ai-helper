package openai

import (
	"encoding/json"

	"github.com/y0ug/ai-helper/pkg/llmclient/types"
)

func MessageToOpenAI(
	m ...*types.ChatMessage,
) []ChatCompletionMessageParam {
	userMessages := make([]ChatCompletionMessageParam, 0)
	for _, msg := range m {
		content := msg.Content[0]
		switch content.Type {
		case types.ContentTypeToolUse:
			// For toolCalls we need to process all of them in one time
			userMessages = append(userMessages, ChatCompletionMessageParam{
				Role:      "assistant",
				ToolCalls: MessageContentToToolCall(msg.Content...),
			})
		case types.ContentTypeToolResult:
			userMessages = append(userMessages, ChatCompletionMessageParam{
				Role:       "tool",
				Content:    content.Content,
				ToolCallID: content.ToolUseID,
			})
		default:
			userMessages = append(userMessages, ChatCompletionMessageParam{
				Role:    msg.Role,
				Content: content.String(),
			})
		}
	}
	return userMessages
}

func ToolCallToMessageContent(t ToolCall) *types.MessageContent {
	// var args map[string]interface{}
	// _ = json.Unmarshal([]byte(t.Function.Arguments), &args)
	return types.NewToolUseContent(t.ID, t.Function.Name, json.RawMessage(t.Function.Arguments))
}

func MessageContentToToolCall(t ...*types.MessageContent) []ToolCall {
	d := make([]ToolCall, 0)
	for _, content := range t {
		if content.Type == types.ContentTypeToolUse {
			d = append(d, ToolCall{
				ID:   content.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      content.Name,
					Arguments: string(content.Input),
				},
			})
		}
	}
	return d
}

func ToolsToOpenAI(tools ...types.Tool) []Tool {
	result := make([]Tool, 0)
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
		aiTool := Tool{
			Type: "function",
			Function: ToolFunction{
				Name:        tool.Name,
				Description: desc,
				Parameters:  tool.InputSchema,
			},
		}
		result = append(result, aiTool)

	}
	return result
}

func ToStopReason(reason string) string {
	match := map[string]string{
		"stop":          "end_turn",
		"length":        "max_tokens",
		"stop_sequence": "stop_sequence", // Stop is same as stop_sequence we dont handle it
		"tool_calls":    "tool_use",
	}
	if r, ok := match[reason]; ok {
		return r
	} else {
		return reason
	}
}

func ToChatResponse(cc *ChatCompletion) *types.ChatResponse {
	cm := &types.ChatResponse{}
	cm.ID = cc.ID
	cm.Model = cc.Model
	cm.Usage = &types.ChatUsage{}
	cm.Usage.InputTokens = cc.Usage.PromptTokens
	cm.Usage.OutputTokens = cc.Usage.CompletionTokens
	for _, choice := range cc.Choices {
		c := types.ChatChoice{}
		for _, call := range choice.Message.ToolCalls {
			c.Content = append(
				c.Content,
				ToolCallToMessageContent(call),
			)
		}

		if choice.Message.Content != "" {
			c.Content = append(c.Content, types.NewTextContent(choice.Message.Content))
		}

		// Role is not choice is our model
		c.Role = choice.Message.Role

		// The reason the model stopped generating tokens. This will be `stop` if the model
		// hit a natural stop point or a provided stop sequence, `length` if the maximum
		// number of tokens specified in the request was reached, `content_filter` if
		// content was omitted due to a flag from our content filters, `tool_calls` if the
		// model called a tool, or `function_call` (deprecated) if the model called a
		// function.
		c.StopReason = ToStopReason(choice.FinishReason)
		cm.Choice = append(cm.Choice, c)
	}
	return cm
}

func ToChatCompletionNewParams(
	params types.ChatParams,
) ChatCompletionNewParams {
	return ChatCompletionNewParams{
		Model:               params.Model,
		MaxCompletionTokens: &params.MaxTokens,
		Temperature:         params.Temperature,
		N:                   params.N,
		Messages:            MessageToOpenAI(params.Messages...),
		Tools:               ToolsToOpenAI(params.Tools...),
	}
}
