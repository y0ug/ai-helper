package conversation

import (
	"context"
	"io"

	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
)

// Manager defines the interface for conversation management
type Manager interface {
	ProcessTurn(ctx context.Context) (*Turn, []*chat.ChatMessage, error)
	UpdateState(newState string) error
	AddMessage(msg *chat.ChatMessage)
	GetCurrentState() string
	GetCurrentTemplate() *Template
	GetHistory() []*chat.ChatMessage
}

// Context defines the interface for context management
type Context interface {
	ExecuteTemplate(templateText string) (string, error)
	SetVariable(key string, value interface{})
	GetVariable(key string) (interface{}, bool)
}

// TurnHandler defines the interface for custom turn processing
type TurnHandler interface {
	PreProcess(ctx context.Context, cm Manager, requestCtx Context) error
	PostProcess(ctx context.Context, cm Manager, response []*chat.ChatResponse, w io.Writer) error
}

// StateTransitioner defines the interface for state transitions
type StateTransitioner interface {
	DetermineNextState(cm Manager, response []*chat.ChatResponse) (string, error)
}

// Turn represents a single conversation interaction
type Turn struct {
	Messages []*chat.ChatMessage
}
