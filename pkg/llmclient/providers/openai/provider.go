package openai

import (
	"context"
	"encoding/json"

	"github.com/y0ug/ai-helper/pkg/llmclient/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/requestoption"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/streaming"
)

type OpenAIProvider struct {
	Client *Client
}

func NewOpenAIProvider(opts ...requestoption.RequestOption) common.LLMProvider {
	return &OpenAIProvider{
		Client: NewClient(opts...),
	}
}

func OpenaAIFinishReasonToStopReason(reason string) string {
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

func OpenaiChatCompletionToChatMessage(cc *ChatCompletion) *common.ChatMessage {
	cm := &common.ChatMessage{}
	cm.ID = cc.ID
	cm.Model = cc.Model
	cm.Usage = &common.ChatMessageUsage{}
	cm.Usage.InputTokens = cc.Usage.PromptTokens
	cm.Usage.OutputTokens = cc.Usage.CompletionTokens
	for _, choice := range cc.Choices {
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

		// The reason the model stopped generating tokens. This will be `stop` if the model
		// hit a natural stop point or a provided stop sequence, `length` if the maximum
		// number of tokens specified in the request was reached, `content_filter` if
		// content was omitted due to a flag from our content filters, `tool_calls` if the
		// model called a tool, or `function_call` (deprecated) if the model called a
		// function.
		c.StopReason = OpenaAIFinishReasonToStopReason(choice.FinishReason)
		cm.Choice = append(cm.Choice, c)
	}
	return cm
}

func BaseChatMessageNewParamsToOpenAI(
	params common.ChatMessageNewParams,
) ChatCompletionNewParams {
	return ChatCompletionNewParams{
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
	params common.ChatMessageNewParams,
) (*common.ChatMessage, error) {
	paramsProvider := BaseChatMessageNewParamsToOpenAI(params)

	resp, err := a.Client.Chat.New(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}

	return OpenaiChatCompletionToChatMessage(&resp), nil
}

func (a *OpenAIProvider) Stream(
	ctx context.Context,
	params common.ChatMessageNewParams,
) (streaming.Streamer[common.EventStream], error) {
	paramsProvider := BaseChatMessageNewParamsToOpenAI(params)

	stream, err := a.Client.Chat.NewStreaming(ctx, paramsProvider)
	if err != nil {
		return nil, err
	}
	return common.NewProviderEventStream(
		stream,
		NewOpenAIEventHandler(),
	), nil
}

func FromLLMToolToOpenAI(tools ...common.Tool) []Tool {
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

func FromOpenaiToolCallToAIContent(t ToolCall) *common.AIContent {
	// var args map[string]interface{}
	// _ = json.Unmarshal([]byte(t.Function.Arguments), &args)
	return common.NewToolUseContent(t.ID, t.Function.Name, json.RawMessage(t.Function.Arguments))
}

func AIContentToolCallsToOpenAI(t ...*common.AIContent) []ToolCall {
	d := make([]ToolCall, 0)
	for _, content := range t {
		if content.Type == common.ContentTypeToolUse {
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

func FromLLMMessageToOpenAi(
	m ...*common.ChatMessageParams,
) []ChatCompletionMessageParam {
	userMessages := make([]ChatCompletionMessageParam, 0)
	for _, msg := range m {
		content := msg.Content[0]
		switch content.Type {
		case common.ContentTypeToolUse:
			// For toolCalls we need to process all of them in one time
			userMessages = append(userMessages, ChatCompletionMessageParam{
				Role:      "assistant",
				ToolCalls: AIContentToolCallsToOpenAI(msg.Content...),
			})
		case common.ContentTypeToolResult:
			userMessages = append(userMessages, ChatCompletionMessageParam{
				Role:       "tool",
				Content:    content.Content,
				ToolCallID: content.ToolUseID,
			})
		default:
			userMessages = append(userMessages, ChatCompletionMessageParam{
				Role:    msg.GetRole(),
				Content: content.String(),
			})
		}
	}
	return userMessages
}

// OpenAIEventHandler processes OpenAI-specific events
type OpenAIEventHandler struct {
	completion ChatCompletion
}

func NewOpenAIEventHandler() *OpenAIEventHandler {
	return &OpenAIEventHandler{}
}

func (h *OpenAIEventHandler) ShouldContinue(chunk ChatCompletionChunk) bool {
	return true
	// return !(chunk.Usage.CompletionTokens != 0 || len(chunk.Choices) == 0)
}

func (h *OpenAIEventHandler) HandleEvent(
	chunk ChatCompletionChunk,
) (common.EventStream, error) {
	h.completion.Accumulate(chunk)
	evt := common.EventStream{Message: OpenaiChatCompletionToChatMessage(&h.completion)}

	if chunk.Usage.CompletionTokens != 0 || len(chunk.Choices) == 0 {
		evt.Type = "message_stop"
		return evt, nil
	}

	evt.Type = "text_delta"
	evt.Delta = chunk.Choices[0].Delta.Content
	return evt, nil
}
