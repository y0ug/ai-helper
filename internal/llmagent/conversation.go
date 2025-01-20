package llmagent

import (
	"context"
	"time"

	"github.com/y0ug/ai-helper/internal/llmcontext"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
)

// ConversationManager handles multi-turn conversations
type ConversationManager struct {
	ID           string
	Templates    map[string]*PromptTemplate
	History      []*Turn
	CurrentState string
	Variables    map[string]interface{}
}

// Turn represents a single conversation turn
type Turn struct {
	TemplateID string
	Input      *llmcontext.RequestContext
	Messages   []*chat.ChatMessage
	Timestamp  time.Time
	State      map[string]interface{}
}

// PromptTemplate defines a template for a conversation state
type PromptTemplate struct {
	ID           string
	SystemPrompt string
	UserPrompt   string
	RequiredVars []string
	NextStates   []string
	Handlers     map[string]TurnHandler
}

// TurnHandler defines the interface for custom turn processing
type TurnHandler interface {
	PreProcess(ctx context.Context, turn *Turn) error
	PostProcess(ctx context.Context, turn *Turn, response []*chat.ChatResponse) error
}

// StateTransitioner defines the interface for state transitions
type StateTransitioner interface {
	DetermineNextState(currentTurn *Turn, response []*chat.ChatResponse) (string, error)
}

// NewConversationManager creates a new conversation manager
func NewConversationManager(id string) *ConversationManager {
	return &ConversationManager{
		ID:        id,
		Templates: make(map[string]*PromptTemplate),
		History:   make([]*Turn, 0),
		Variables: make(map[string]interface{}),
	}
}
