package ai

import (
	"fmt"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/prompt"
)

// Agent represents an AI conversation agent that maintains state and history
type Agent struct {
	ID           string           // Unique identifier for this agent/session
	Model        *Model           // The AI model being used
	Messages     []Message        // Conversation history
	Command      *config.Command  // Current active command
	TemplateData *prompt.TemplateData // Data for template processing
}

// LoadCommand loads a command configuration into the agent
func (a *Agent) LoadCommand(cmd *config.Command) error {
	a.Command = cmd
	
	// Load environment variables
	a.TemplateData.LoadEnvironment()
	
	// Load any required files
	if len(cmd.Files) > 0 {
		if err := a.TemplateData.LoadFiles(cmd.Files); err != nil {
			return fmt.Errorf("failed to load command files: %w", err)
		}
	}
	
	// Process system message template if present
	if cmd.System != "" {
		systemMsg, err := prompt.Execute(cmd.System, a.TemplateData)
		if err != nil {
			return fmt.Errorf("failed to process system template: %w", err)
		}
		a.AddSystemMessage(systemMsg)
	}
	
	return nil
}

// ApplyCommand applies the loaded command's prompt with the given input
func (a *Agent) ApplyCommand(input string) error {
	if a.Command == nil {
		return fmt.Errorf("no command loaded")
	}

	// Update template data with new input
	a.TemplateData.Input = input

	// Process the prompt template
	processedPrompt, err := prompt.Execute(a.Command.Prompt, a.TemplateData)
	if err != nil {
		return fmt.Errorf("failed to process prompt template: %w", err)
	}

	a.AddMessage("user", processedPrompt)
	return nil
}

// NewAgent creates a new Agent instance
func NewAgent(id string, model *Model) *Agent {
	return &Agent{
		ID:           id,
		Model:        model,
		Messages:     make([]Message, 0),
		TemplateData: prompt.NewTemplateData(""),
	}
}

// AddMessage adds a new message to the agent's conversation history
func (a *Agent) AddMessage(role, content string) {
	a.Messages = append(a.Messages, Message{
		Role:    role,
		Content: content,
	})
}

// GetMessages returns the current message history
func (a *Agent) GetMessages() []Message {
	return a.Messages
}

// AddSystemMessage adds a system message to the start of the conversation
func (a *Agent) AddSystemMessage(content string) {
	// If first message is already a system message, replace it
	if len(a.Messages) > 0 && a.Messages[0].Role == "system" {
		a.Messages[0].Content = content
		return
	}
	
	// Insert system message at the beginning
	a.Messages = append([]Message{{Role: "system", Content: content}}, a.Messages...)
}
