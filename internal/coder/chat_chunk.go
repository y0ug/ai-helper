package coder

import "github.com/y0ug/ai-helper/internal/coder/prompts"

type ChatChunks struct {
	System        []prompts.Message
	Examples      []prompts.Message
	Done          []prompts.Message
	Repo          []prompts.Message
	ReadOnlyFiles []prompts.Message
	ChatFiles     []prompts.Message
	Cur           []prompts.Message
	Reminder      []prompts.Message
}

func (c *ChatChunks) AllMessages() []prompts.Message {
	var messages []prompts.Message
	messages = append(messages, c.System...)
	messages = append(messages, c.Examples...)
	messages = append(messages, c.Done...)
	messages = append(messages, c.Repo...)
	messages = append(messages, c.ReadOnlyFiles...)
	messages = append(messages, c.ChatFiles...)
	messages = append(messages, c.Cur...)
	messages = append(messages, c.Reminder...)
	return messages
}

func (c *ChatChunks) AddCacheControlHeaders() {
	// Add cache control headers to messages if needed
	// Implementation depends on your caching strategy
}
