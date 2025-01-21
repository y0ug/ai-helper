package conversation

import (
	"context"
	"io"

	ctmg "github.com/y0ug/ai-helper/pkg/conversation/context"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
)

// Manager defines the interface for conversation management
type Manager interface {
	ProcessTurn(ctx context.Context) ([]*chat.ChatMessage, error)
	UpdateState(newState string) error
	AddMessage(msg ...*chat.ChatMessage)
	GetCurrentState() string
	GetCurrentTemplate() *Template
	GetHistory() []*chat.ChatMessage
	GetCtx() ctmg.ContextManager
	IsInputNeeded() bool
}

// TurnHandler defines the interface for custom turn processing
type TurnHandler interface {
	PreProcess(ctx context.Context, cm Manager) error
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
