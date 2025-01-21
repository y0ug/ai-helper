package conversation

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"slices"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/filemanager"
	ctmg "github.com/y0ug/ai-helper/pkg/conversation/context"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
)

// ConversationManager handles multi-turn conversations
type ConversationManager struct {
	ID           string
	logger       *slog.Logger
	Templates    map[string]*Template
	History      []*chat.ChatMessage
	CurrentState string
	ctx          ctmg.ContextManager
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
		if _, exists := cm.ctx.GetVariable(required); !exists {
			return fmt.Errorf("missing required variable: %s", required)
		}
	}
	return nil
}

// NewConversationManager creates a new conversation manager
func NewConversationManager(
	id string,
	logger *slog.Logger,
	cmd *config.Command,
) *ConversationManager {
	if logger == nil {
		logger = slog.New(slog.Default().Handler()).WithGroup("conversation_manager")
	}

	fm := filemanager.NewLocalFileManager()
	ctx := ctmg.NewContext(cmd, fm)
	cm := &ConversationManager{
		ID:        id,
		logger:    logger,
		Templates: make(map[string]*Template),
		History:   make([]*chat.ChatMessage, 0),
		ctx:       ctx,
	}

	cm.loadCommand(cmd)
	return cm
}

func (cm *ConversationManager) GetCtx() ctmg.ContextManager {
	return cm.ctx
}

// LoadCommand loads a command configuration into the conversation manager
func (cm *ConversationManager) loadCommand(command *config.Command) error {
	// Convert command templates to PromptTemplates
	for id, tmpl := range command.Templates {
		template := &Template{
			ID:           id,
			SystemPrompt: tmpl.System,
			UserPrompt:   tmpl.Prompt,
			Variables:    tmpl.Variables,
			RequiredVars: extractRequiredVars(tmpl.Variables),
			NextStates:   tmpl.NextStates,
			Handlers:     make(map[string]TurnHandler),
			PreTurnCmds:  tmpl.PreTurnCmds,
			PostTurnCmds: tmpl.PostTurnCmds,
		}

		// Register handlers
		for _, handlerName := range tmpl.Handlers {
			switch handlerName {
			case "codediff":
				template.Handlers[handlerName] = NewCodeDiffHandler()
			}
		}
		cm.Templates[template.ID] = template
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

func (cm *ConversationManager) IsInputNeeded() bool {
	k := "Input"
	if slices.Contains(cm.GetCurrentTemplate().RequiredVars, k) {
		if _, ok := cm.ctx.GetVariable(k); ok {
			return false
		}
		return true
	}
	return false
}

func (cm *ConversationManager) SetInput(input string) error {
	if input != "" {
		cm.ctx.SetVariable("Input", input)
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
) ([]*chat.ChatMessage, error) {
	template, exists := cm.Templates[cm.CurrentState]
	if !exists {
		return nil, fmt.Errorf("no template found for current state: %s", cm.CurrentState)
	}

	// Execute pre-turn commands
	for _, cmdStr := range template.PreTurnCmds {
		output, err := cm.executeCommand(cmdStr)
		if err != nil {
			return nil, fmt.Errorf("pre-turn command failed: %w", err)
		}
		cm.ctx.SetVariable("PreTurnOutput_"+cmdStr, output)
	}

	// Run pre-processing handlers
	for _, handler := range template.Handlers {
		if err := handler.PreProcess(ctx, cm); err != nil {
			return nil, fmt.Errorf("pre-process error: %w", err)
		}
	}

	if err := cm.ValidateRequiredVars(); err != nil {
		return nil, fmt.Errorf("variable validation failed: %w", err)
	}

	// Generate messages for this turn
	messages, err := cm.generateMessages()
	if err != nil {
		return nil, fmt.Errorf("failed to generate messages: %w", err)
	}

	// generateMesssages include already the history
	// since it can overide the system prompt
	// so we rewrite the history
	cm.History = messages

	// // Run post-processing handlers
	// for _, handler := range template.Handlers {
	// 	if err := handler.PostProcess(ctx, cm, nil); err != nil {
	// 		return nil, nil, fmt.Errorf("post-process error: %w", err)
	// 	}
	// }

	// // Execute post-turn commands
	// for _, cmdStr := range template.PostTurnCmds {
	// 	output, err := cm.executeCommand(cmdStr)
	// 	if err != nil {
	// 		return nil, nil, fmt.Errorf("post-turn command failed: %w", err)
	// 	}
	// 	requestCtx.Vars["PostTurnOutput_"+cmdStr] = output
	// }

	return messages, nil
}

func (cm *ConversationManager) GetCurrentState() string {
	return cm.CurrentState
}

func (cm *ConversationManager) UpdateState(newState string) error {
	if slices.Contains(cm.GetCurrentTemplate().NextStates, newState) {
		cm.CurrentState = newState
		return nil
	}
	return fmt.Errorf("invalid state transition: %s -> %s", cm.CurrentState, newState)
}

// generateMessages creates the message sequence for a turn
func (cm *ConversationManager) generateMessages() ([]*chat.ChatMessage, error) {
	var messages []*chat.ChatMessage
	template := cm.GetCurrentTemplate()

	// Generate system message if provided
	if template.SystemPrompt != "" {
		systemContent, err := cm.ctx.ExecuteTemplate(template.ID, template.SystemPrompt)
		if err != nil {
			return nil, fmt.Errorf("failed to execute system template: %w", err)
		}
		messages = append(messages, chat.NewMessage("system", chat.NewTextContent(systemContent)))
	}

	// Add relevant conversation history from previous turns
	messages = append(messages, cm.History...)

	// Generate user message from prompt template
	promptContent, err := cm.ctx.ExecuteTemplate(template.ID, template.UserPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to execute prompt template: %w", err)
	}
	messages = append(messages, chat.NewMessage("user", chat.NewTextContent(promptContent)))

	return messages, nil
}

func (cm *ConversationManager) AddMessage(msg ...*chat.ChatMessage) {
	cm.History = append(cm.History, msg...)
}

// GetCurrentTemplate returns the currently active template
func (cm *ConversationManager) GetCurrentTemplate() *Template {
	return cm.Templates[cm.CurrentState]
}

// GetHistory returns the conversation history
func (cm *ConversationManager) GetHistory() []*chat.ChatMessage {
	return cm.History
}
