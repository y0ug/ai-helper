package conversation

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"slices"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/filemanager"
	"github.com/y0ug/ai-helper/internal/llmcontext"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
)

// ConversationManager handles multi-turn conversations
type ConversationManager struct {
	ID           string
	logger       *slog.Logger
	Command      *config.Command
	Templates    map[string]*PromptTemplate
	History      []*chat.ChatMessage
	CurrentState string
	Variables    map[string]interface{}
	FileManager  filemanager.FileManager
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
func (cm *ConversationManager) AddFile(path string, readOnly bool) error {
	return cm.FileManager.AddFile(path, readOnly)
}

func (cm *ConversationManager) RemoveFile(path string) error {
	return cm.FileManager.RemoveFile(path)
}

// PromptTemplate defines a template for a conversation state
type PromptTemplate struct {
	ID           string
	SystemPrompt string
	UserPrompt   string
	RequiredVars []string
	NextStates   []string
	Handlers     map[string]TurnHandler
	PreTurnCmds  []string
	PostTurnCmds []string
}

// NewConversationManager creates a new conversation manager
func NewConversationManager(id string, logger *slog.Logger) *ConversationManager {
	if logger == nil {
		logger = slog.New(slog.Default().Handler()).WithGroup("conversation_manager")
	}
	logger.Info("Creating new conversation manager", "id", id)
	fm := filemanager.NewLocalFileManager()
	return &ConversationManager{
		ID:          id,
		logger:      logger,
		Templates:   make(map[string]*PromptTemplate),
		History:     make([]*chat.ChatMessage, 0),
		Variables:   make(map[string]interface{}),
		FileManager: fm,
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

		// Register handlers
		for _, handlerName := range tmpl.Handlers {
			switch handlerName {
			case "codediff":
				template.Handlers[handlerName] = NewCodeDiffHandler()
			}
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
	if input != "" {
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

	// Create a map of file contents for the request context
	filesInCtx := cm.FileManager.GetFiles()
	files := make(map[string]string)
	for path := range filesInCtx {
		// Get both status and changed state
		status, err := cm.FileManager.GetFileStatus(path)
		if err != nil {
			return nil, nil, fmt.Errorf("error checking file status for %s: %w", path, err)
		}

		changed, err := cm.FileManager.HasFileChanged(path)
		if err != nil {
			return nil, nil, fmt.Errorf("error checking file changes for %s: %w", path, err)
		}

		// Add file to context if:
		// 1. It has changed since last send OR
		// 2. It is out of sync with disk OR
		// 3. It has been modified
		if changed || status == filemanager.StatusOutOfSync ||
			status == filemanager.StatusModified {
			content, isEditable, err := cm.FileManager.GetFileContent(path)
			if err != nil {
				return nil, nil, fmt.Errorf("error getting content for %s: %w", path, err)
			}

			files[path] = content
			requestCtx.Vars[path+"_readonly"] = !isEditable
			// requestCtx.Vars[path+"_status"] = status.String()

			// Mark as sent only if we successfully added it to context
			if err := cm.FileManager.MarkFileAsSent(path); err != nil {
				return nil, nil, fmt.Errorf("error marking %s as sent: %w", path, err)
			}
		}
	}
	requestCtx.Files = files

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

// UpdateFileFromLLMResponse updates a file with content from LLM response
func (cm *ConversationManager) UpdateFileFromLLMResponse(path string, content string) error {
	// Check if file exists and is editable
	_, isEditable, err := cm.FileManager.GetFileContent(path)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	if !isEditable {
		return fmt.Errorf("file %s is read-only", path)
	}

	return cm.FileManager.UpdateFileContent(path, content)
}

// GetFileStatus gets the status of a file
func (cm *ConversationManager) GetFileStatus(path string) (filemanager.FileStatus, error) {
	return cm.FileManager.GetFileStatus(path)
}

// Optional: Add method to check if files have changed
func (cm *ConversationManager) CheckFilesStatus() map[string]filemanager.FileStatus {
	statuses := make(map[string]filemanager.FileStatus)

	// Check all files mentioned in variables
	for path := range cm.Variables {
		if status, err := cm.FileManager.GetFileStatus(path); err == nil {
			statuses[path] = status
		}
	}

	return statuses
}
