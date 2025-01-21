package conversation

import (
	"context"
	"fmt"
	"io"
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

// ValidateRequiredVars checks if all required variables are present
func (cm *ConversationManager) ValidateRequiredVars() error {
	requiredVars := cm.GetCurrentTemplate().RequiredVars
	for _, required := range requiredVars {
		if _, exists := cm.GetCtx().GetVariable(required); !exists {
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

	err := cm.loadCommand(cmd)
	if err != nil {
		return nil //, fmt.Errorf("failed to load command: %w", err)
	}
	return cm
}

func (cm *ConversationManager) GetCtx() ctmg.ContextManager {
	return cm.ctx
}

// LoadCommand loads a command configuration into the conversation manager
func (cm *ConversationManager) loadCommand(command *config.Command) error {
	// Convert command templates to PromptTemplates
	cm.Templates = NewTemplatesFromConfig(command.Templates)

	// Set initial state
	initialState := command.InitialState
	if initialState == "" {
		// If no initial state specified, use the first template
		for id := range command.Templates {
			initialState = id
			break
		}
	}

	if _, ok := cm.Templates[initialState]; !ok {
		return fmt.Errorf("initial state not found: %s", initialState)
	}
	cm.CurrentState = initialState

	// We can't call updateState because GetTemplate() is not working yet
	// if err := cm.UpdateState(initialState); err != nil {
	// 	return fmt.Errorf("failed to update state: %w", err)
	// }

	return nil
}

func (cm *ConversationManager) IsInputNeeded() bool {
	k := "Input"
	if slices.Contains(cm.GetCurrentTemplate().RequiredVars, k) {
		if _, ok := cm.GetCtx().GetVariable(k); ok {
			// Input already set
			return false
		}
		return true
	}
	return false
}

func (cm *ConversationManager) SetInput(input string) error {
	if input != "" {
		cm.GetCtx().SetVariable("Input", input)
	}
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
		cm.GetCtx().SetVariable("PreTurnOutput_"+cmdStr, output)
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

	return messages, nil
}

// In pkg/conversation/manager.go
func (cm *ConversationManager) ProcessResponse(
	ctx context.Context,
	responses []*chat.ChatResponse,
	w io.Writer,
) error {
	if len(responses) == 0 {
		return fmt.Errorf("no responses to process")
	}

	template := cm.GetCurrentTemplate()

	// Run post-processing handlers
	for _, handler := range template.Handlers {
		if err := handler.PostProcess(ctx, cm, responses, w); err != nil {
			return fmt.Errorf("post-process error: %w", err)
		}
	}

	// Add responses to conversation history
	for _, resp := range responses {
		cm.AddMessage(resp.ToMessageParams())
	}

	// Handle state transition
	nextState, err := NewDefaultStateTransitioner().DetermineNextState(cm, responses)
	if err != nil {
		cm.logger.Warn("State transition error", "error", err)
	} else {
		currentState := cm.GetCurrentState()
		if nextState != currentState {
			cm.logger.Info("State transition", "from", currentState, "to", nextState)
			if err := cm.UpdateState(nextState); err != nil {
				cm.logger.Error("Failed to update state", "error", err)
			}
		}
	}

	// Execute post-turn commands
	template = cm.GetCurrentTemplate()
	for _, cmdStr := range template.PostTurnCmds {
		output, err := cm.executeCommand(cmdStr)
		if err != nil {
			return fmt.Errorf("post-turn command failed: %w", err)
		}
		cm.GetCtx().SetVariable("PostTurnOutput_"+cmdStr, output)
	}

	return nil
}

// generateMessages creates the message sequence for a turn
func (cm *ConversationManager) generateMessages() ([]*chat.ChatMessage, error) {
	var messages []*chat.ChatMessage
	template := cm.GetCurrentTemplate()

	// Add relevant conversation history from previous turns
	// we should not apply system messages if that the case
	messages = cm.History
	// messages = append(messages, cm.History...)

	for _, msg := range template.Messages {
		content, err := cm.GetCtx().ExecuteTemplate(template.ID, msg.Content)
		if err != nil {
			err := fmt.Errorf("failed to execute message template: %w", err)
			cm.logger.Error(
				"Failed to execute message template",
				"error",
				err,
				"template.ID",
				template.ID,
				"role",
				msg.Role,
				"content",
				msg.Content,
			)
			continue
		}
		messages = append(messages, chat.NewMessage(msg.Role, chat.NewTextContent(content)))
	}

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
