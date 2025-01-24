package assistant

import (
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type ChatChunks struct {
	System        []*chat.ChatMessage
	Examples      []*chat.ChatMessage
	Done          []*chat.ChatMessage
	Repo          []*chat.ChatMessage
	ReadOnlyFiles []*chat.ChatMessage
	ChatFiles     []*chat.ChatMessage
	Cur           []*chat.ChatMessage
	Reminder      []*chat.ChatMessage
}

func (c *ChatChunks) AllMessages() []*chat.ChatMessage {
	var messages []*chat.ChatMessage
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
	// c.AddCacheControl(c.System)
	// c.AddCacheControl(c.ReadOnlyFiles)
	// c.AddCacheControl(c.ChatFiles)
	// c.AddCacheControl(c.Done)
	// c.AddCacheControl(c.Examples)

	// 4 max cache messages for anthropic
	if len(c.Examples) > 0 {
		c.AddCacheControl(c.Examples)
	} else {
		c.AddCacheControl(c.System)
	}

	if len(c.Repo) > 0 {
		c.AddCacheControl(c.Repo)
	} else {
		c.AddCacheControl(c.ReadOnlyFiles)
	}

	c.AddCacheControl(c.ChatFiles)
}

func (c *ChatChunks) AddCacheControl(messages []*chat.ChatMessage) {
	for _, msg := range messages {
		msg.SetCache()
	}
}
