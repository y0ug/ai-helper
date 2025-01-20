package llmagent

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/llmcontext"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
)

// ConversationManager handles multi-turn conversations
type ConversationManager struct {
	ID           string
	Command      *config.Command
	Templates    map[string]*PromptTemplate
	History      []*chat.ChatMessage
	CurrentState string
	Variables    map[string]interface{}
	Files        map[string]string
}

// NewTurn creates a new turn with initialized maps
// func NewTurn(templateID string, template *PromptTemplate) *Turn {
// 	return &Turn{
// 		TemplateID: templateID,
// 		Template:   template,
// 		Timestamp:  time.Now(),
// 		Variables:  make(map[string]interface{}),
// 		Files:      make(map[string]string),
// 	}
// }

// ValidateRequiredVars checks if all required variables are present
func (cm *ConversationManager) ValidateRequiredVars() error {
	requiredVars := cm.GetCurrentTemplate().RequiredVars
	for _, required := range requiredVars {
		if _, exists := cm.Variables[required]; !exists {
			return fmt.Errorf("missing required variable: %s", required)
		}
	}
	return nil
}

// SetVariable sets a variable for the turn
func (cm *ConversationManager) SetVariable(name string, value interface{}) {
	cm.Variables[name] = value
}

// LoadFile loads a file into the turn context
func (cm *ConversationManager) LoadFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", path, err)
	}
	cm.Files[path] = string(content)
	return nil
}

// PromptTemplate defines a template for a conversation state
type PromptTemplate struct {
	ID              string
	SystemPrompt    string
	UserPrompt      string
	RequiredVars    []string
	NextStates      []string
	Handlers        map[string]TurnHandler
	PreTurnCmds     []string
	PostTurnCmds    []string
}

// TurnHandler defines the interface for custom turn processing
type TurnHandler interface {
	PreProcess(
		ctx context.Context,
		cm *ConversationManager,
		requestCtx *llmcontext.RequestContext,
	) error
	PostProcess(ctx context.Context, cm *ConversationManager, response []*chat.ChatResponse) error
}

// StateTransitioner defines the interface for state transitions
type StateTransitioner interface {
	DetermineNextState(cm *ConversationManager, response []*chat.ChatResponse) (string, error)
}

// NewConversationManager creates a new conversation manager
func NewConversationManager(id string) *ConversationManager {
	return &ConversationManager{
		ID:        id,
		Templates: make(map[string]*PromptTemplate),
		History:   make([]*chat.ChatMessage, 0),
		Variables: make(map[string]interface{}),
		Files:     make(map[string]string),
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
			PreTurnCmds:  tmpl.PreTurnCmds,
			PostTurnCmds: tmpl.PostTurnCmds,
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
		cm.Variables["Input"] = input
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

type Turn struct {
	Messages []*chat.ChatMessage
}

// executeCommand runs a shell command and returns its output
func (cm *ConversationManager) executeCommand(cmdStr string) (string, error) {
	cmd := exec.Command("sh", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command failed: %s: %w", cmdStr, err)
	}
	return string(output), nil
}

// ProcessTurn handles a single conversation turn
func (cm *ConversationManager) ProcessTurn(
	ctx context.Context,
) (*Turn, []*chat.ChatMessage, error) {
	template, exists := cm.Templates[cm.CurrentState]
	turn := &Turn{}
	if !exists {
		return nil, nil, fmt.Errorf("no template found for current state: %s", cm.CurrentState)
	}
	requestCtx := llmcontext.NewRequestContext(cm.Command)

	// Copy current variables and files to the turn
	requestCtx.Vars = cm.Variables
	requestCtx.Files = cm.Files

	// Execute pre-turn commands
	for _, cmdStr := range template.PreTurnCmds {
		output, err := cm.executeCommand(cmdStr)
		if err != nil {
			return nil, nil, fmt.Errorf("pre-turn command failed: %w", err)
		}
		requestCtx.Vars["PreTurnOutput_"+cmdStr] = output
	}

	// Run pre-processing handlers
	for _, handler := range template.Handlers {
		if err := handler.PreProcess(ctx, cm, requestCtx); err != nil {
			return nil, nil, fmt.Errorf("pre-process error: %w", err)
		}
	}

	if err := cm.ValidateRequiredVars(); err != nil {
		return nil, nil, fmt.Errorf("variable validation failed: %w", err)
	}

	// Generate messages for this turn
	messages, err := cm.generateMessages(turn, requestCtx, template)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate messages: %w", err)
	}

	turn.Messages = messages
	cm.History = append(cm.History, messages...)

	// Run post-processing handlers
	for _, handler := range template.Handlers {
		if err := handler.PostProcess(ctx, cm, nil); err != nil {
			return nil, nil, fmt.Errorf("post-process error: %w", err)
		}
	}

	// Execute post-turn commands
	for _, cmdStr := range template.PostTurnCmds {
		output, err := cm.executeCommand(cmdStr)
		if err != nil {
			return nil, nil, fmt.Errorf("post-turn command failed: %w", err)
		}
		requestCtx.Vars["PostTurnOutput_"+cmdStr] = output
	}

	return turn, messages, nil
}

// generateMessages creates the message sequence for a turn
func (cm *ConversationManager) generateMessages(
	turn *Turn,
	requestCtx *llmcontext.RequestContext,
	template *PromptTemplate,
) ([]*chat.ChatMessage, error) {
	var messages []*chat.ChatMessage

	// Generate system message if provided
	if template.SystemPrompt != "" {
		systemContent, err := requestCtx.Execute(template.SystemPrompt)
		if err != nil {
			return nil, fmt.Errorf("failed to execute system template: %w", err)
		}
		messages = append(messages, chat.NewMessage("system", chat.NewTextContent(systemContent)))
	}

	// Add relevant conversation history from previous turns
	for _, msg := range cm.History {
		messages = append(messages, msg)
	}

	// Generate user message from prompt template
	promptContent, err := requestCtx.Execute(template.UserPrompt)
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
func (cm *ConversationManager) GetHistory() []*chat.ChatMessage {
	return cm.History
}
