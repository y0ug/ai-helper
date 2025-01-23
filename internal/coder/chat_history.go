package coder

import (
	"github.com/y0ug/ai-helper/internal/coder/prompts"
)

type ChatHistory struct {
	curMessages  []prompts.Message
	doneMessages []prompts.Message
}

func NewChatHistory() *ChatHistory {
	return &ChatHistory{
		curMessages:  make([]prompts.Message, 0),
		doneMessages: make([]prompts.Message, 0),
	}
}

func (ch *ChatHistory) AddMessage(msg prompts.Message) {
	ch.curMessages = append(ch.curMessages, msg)
}

func (ch *ChatHistory) GetCurrentMessages() []prompts.Message {
	return ch.curMessages
}

func (ch *ChatHistory) GetDoneMessages() []prompts.Message {
	return ch.doneMessages
}

func (ch *ChatHistory) MoveCurrentToDone(responseMsg string) {
	ch.doneMessages = append(ch.doneMessages, ch.curMessages...)
	ch.curMessages = make([]prompts.Message, 0)

	if responseMsg != "" {
		ch.doneMessages = append(ch.doneMessages,
			prompts.Message{
				Role:    "user",
				Content: responseMsg,
			},
			prompts.Message{
				Role:    "assistant", 
				Content: "Ok.",
			},
		)
	}
}
