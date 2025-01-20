package llmagent

import (
	"context"
	"fmt"
	"slices"
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
	Template   *PromptTemplate
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

// AddTemplate adds a new template to the conversation manager
func (cm *ConversationManager) AddTemplate(template *PromptTemplate) error {
	if template.ID == "" {
		return fmt.Errorf("template ID cannot be empty")
	}
	cm.Templates[template.ID] = template
	return nil
}

func (cm *ConversationManager) IsInputNeeded() bool {
	k := "Input"
	if slices.Contains(cm.GetCurrentTemplate().RequiredVars, k) {
		if _, ok := cm.Variables[k]; ok {
			return false
		}
		return true
	}
	return false
}

// StartConversation initializes a conversation with a specific template
func (cm *ConversationManager) StartConversation(templateID string) error {
	if _, exists := cm.Templates[templateID]; !exists {
		return fmt.Errorf("template %s not found", templateID)
	}
	cm.CurrentState = templateID
	return nil
}

// ProcessTurn handles a single conversation turn
func (cm *ConversationManager) ProcessTurn(
	ctx context.Context,
	input *llmcontext.RequestContext,
) (*Turn, error) {
	template, exists := cm.Templates[cm.CurrentState]
	if !exists {
		return nil, fmt.Errorf("no template found for current state: %s", cm.CurrentState)
	}

	turn := &Turn{
		TemplateID: cm.CurrentState,
		Template:   template,
		Input:      input,
		Timestamp:  time.Now(),
		State:      make(map[string]interface{}),
	}

	// Run pre-processing handlers
	for _, handler := range template.Handlers {
		if err := handler.PreProcess(ctx, turn); err != nil {
			return nil, fmt.Errorf("pre-process error: %w", err)
		}
	}

	cm.History = append(cm.History, turn)
	return turn, nil
}

// GetCurrentTemplate returns the currently active template
func (cm *ConversationManager) GetCurrentTemplate() *PromptTemplate {
	return cm.Templates[cm.CurrentState]
}

// GetHistory returns the conversation history
func (cm *ConversationManager) GetHistory() []*Turn {
	return cm.History
}
