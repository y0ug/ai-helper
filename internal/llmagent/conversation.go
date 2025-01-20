package llmagent

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/llmcontext"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
)

// ConversationManager handles multi-turn conversations
type ConversationManager struct {
	ID           string
	Command      *config.Command
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

// LoadCommand loads a command configuration into the conversation manager
func (cm *ConversationManager) LoadCommand(command *config.Command) error {
	cm.Command = command
	// Convert command templates to PromptTemplates
	for id, tmpl := range command.Templates {
		template := &PromptTemplate{
			ID:           id,
			SystemPrompt: tmpl.System,
			UserPrompt:   tmpl.Prompt,
			RequiredVars: extractRequiredVars(tmpl.Variables),
			NextStates:   tmpl.NextStates,
			Handlers:     make(map[string]TurnHandler),
		}

		if err := cm.AddTemplate(template); err != nil {
			return fmt.Errorf("failed to add template %s: %w", id, err)
		}
	}

	// Set initial state
	initialState := command.InitialState
	if initialState == "" {
		// If no initial state specified, use the first template
		for id := range command.Templates {
			initialState = id
			break
		}
	}

	if err := cm.StartConversation(initialState); err != nil {
		return fmt.Errorf("failed to start conversation: %w", err)
	}

	return nil
}

// extractRequiredVars extracts required variable names from Variable slice
func extractRequiredVars(vars []config.Variable) []string {
	required := make([]string, 0)
	for _, v := range vars {
		if v.Type != "" {
			required = append(required, v.Name)
		}
	}
	return required
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

func (cm *ConversationManager) SetInput(input string) error {
	if cm.IsInputNeeded() {
		// .logger.Debug("adding new input", "input", input)
		cm.ProcessTurn().Variables["Input"] = input
	}
	return nil
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
) (*Turn, []*chat.ChatMessage, error) {
	template, exists := cm.Templates[cm.CurrentState]
	if !exists {
		return nil, nil, fmt.Errorf("no template found for current state: %s", cm.CurrentState)
	}

	turn := &Turn{
		TemplateID: cm.CurrentState,
		Template:   template,
		Timestamp:  time.Now(),
		State:      make(map[string]interface{}),
	}

	// Run pre-processing handlers
	for _, handler := range template.Handlers {
		if err := handler.PreProcess(ctx, turn); err != nil {
			return nil, nil, fmt.Errorf("pre-process error: %w", err)
		}
	}

	// Generate messages for this turn
	messages, err := cm.generateMessages(turn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate messages: %w", err)
	}

	turn.Messages = messages
	cm.History = append(cm.History, turn)

	// Run post-processing handlers
	for _, handler := range template.Handlers {
		if err := handler.PostProcess(ctx, turn, nil); err != nil {
			return nil, nil, fmt.Errorf("post-process error: %w", err)
		}
	}

	return turn, messages, nil
}

// generateMessages creates the message sequence for a turn
func (cm *ConversationManager) generateMessages(turn *Turn) ([]*chat.ChatMessage, error) {
	var messages []*chat.ChatMessage

	// Generate system message if provided
	if turn.Template.SystemPrompt != "" {
		systemContent, err := turn.Input.Execute(turn.Template.SystemPrompt)
		if err != nil {
			return nil, fmt.Errorf("failed to execute system template: %w", err)
		}
		messages = append(messages, chat.NewMessage("system", chat.NewTextContent(systemContent)))
	}

	// Add conversation history from previous turns
	for _, prevTurn := range cm.History {
		if prevTurn.TemplateID == turn.TemplateID {
			continue // Skip current turn
		}
		messages = append(messages, prevTurn.Messages...)
	}

	// Generate user message from prompt template
	promptContent, err := turn.Input.Execute(turn.Template.UserPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to execute prompt template: %w", err)
	}
	messages = append(messages, chat.NewMessage("user", chat.NewTextContent(promptContent)))

	return messages, nil
}

// GetCurrentTemplate returns the currently active template
func (cm *ConversationManager) GetCurrentTemplate() *PromptTemplate {
	return cm.Templates[cm.CurrentState]
}

// GetHistory returns the conversation history
func (cm *ConversationManager) GetHistory() []*Turn {
	return cm.History
}

// LoadFiles loads files into the conversation context
func (cm *ConversationManager) LoadFiles(paths ...string) error {
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading file %s: %w", path, err)
		}
		cm.Variables[path] = string(content)
	}
	return nil
}

// RemoveFiles removes files from the conversation context
func (cm *ConversationManager) RemoveFiles(paths ...string) {
	for _, path := range paths {
		delete(cm.Variables, path)
	}
}

// GetLoadedFiles returns a list of currently loaded files
func (cm *ConversationManager) GetLoadedFiles() []string {
	files := make([]string, 0)
	for k := range cm.Variables {
		if strings.HasPrefix(k, "/") || strings.HasPrefix(k, "./") {
			files = append(files, k)
		}
	}
	return files
}
