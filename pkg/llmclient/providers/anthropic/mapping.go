package anthropic

import "github.com/y0ug/ai-helper/pkg/llmclient/chat"

func BaseChatMessageNewParamsToAnthropic(
	params chat.ChatParams,
) MessageNewParams {
	systemPromt := ""
	msgs := make([]MessageParam, 0)
	for _, m := range params.Messages {
		if m.Role == "system" {
			systemPromt = m.Content[0].String()
			continue
		}
		msgs = append(msgs, MessageParam{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	paramsProvider := MessageNewParams{
		Model:       params.Model,
		MaxTokens:   params.MaxTokens,
		Temperature: params.Temperature,
		Messages:    msgs,
		System:      systemPromt,
		Tools:       params.Tools,
	}
	return paramsProvider
}

func AnthropicMessageToChatMessage(am *Message) *chat.ChatResponse {
	cm := &chat.ChatResponse{}
	cm.ID = am.ID
	cm.Model = am.Model
	cm.Usage = &chat.ChatUsage{}
	cm.Usage.InputTokens = am.Usage.InputTokens
	cm.Usage.OutputTokens = am.Usage.OutputTokens

	c := chat.ChatChoice{}
	c.Content = append(c.Content, am.Content...)
	c.Role = am.Role
	c.StopReason = am.StopReason
	cm.Choice = append(cm.Choice, c)
	return cm
}