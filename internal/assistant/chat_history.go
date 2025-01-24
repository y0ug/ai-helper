package assistant

import "github.com/y0ug/ai-helper/pkg/llmhaven/chat"

type ChatHistory struct {
	curMessages  []*chat.ChatMessage
	doneMessages []*chat.ChatMessage
}

func NewChatHistory() *ChatHistory {
	return &ChatHistory{
		curMessages:  make([]*chat.ChatMessage, 0),
		doneMessages: make([]*chat.ChatMessage, 0),
	}
}

func (ch *ChatHistory) AddMessage(msg *chat.ChatMessage) {
	ch.curMessages = append(ch.curMessages, msg)
}

func (ch *ChatHistory) GetCurrentMessages() []*chat.ChatMessage {
	return ch.curMessages
}

func (ch *ChatHistory) GetDoneMessages() []*chat.ChatMessage {
	return ch.doneMessages
}

func (ch *ChatHistory) MoveCurrentToDone(responseMsg string) {
	ch.doneMessages = append(ch.doneMessages, ch.curMessages...)
	ch.curMessages = make([]*chat.ChatMessage, 0)

	if responseMsg != "" {
		ch.doneMessages = append(
			ch.doneMessages,
			chat.NewMessage("user", chat.NewTextContent(responseMsg)),
			chat.NewMessage("assistant", chat.NewTextContent("Ok.")),
		)
	}
}
