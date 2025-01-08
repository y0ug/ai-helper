package ai

import (
	"fmt"

	"github.com/y0ug/ai-helper/internal/config"
)

// Agent represents an AI conversation agent that maintains state and history
type Agent struct {
	ID       string           // Unique identifier for this agent/session
	Model    *Model           // The AI model being used
	Messages []Message        // Conversation history
	Command  *config.Command  // Current active command
}

// LoadCommand loads a command configuration into the agent
func (a *Agent) LoadCommand(cmd *config.Command) {
	a.Command = cmd
	
	// If there's a system message, apply it
	if cmd.System != "" {
		a.AddSystemMessage(cmd.System)
	}
}

// ApplyCommand applies the loaded command's prompt with the given input
func (a *Agent) ApplyCommand(input string) {
	if a.Command == nil {
		return
	}

	// If the command expects input, format the prompt with it
	if a.Command.Input {
		a.AddMessage("user", fmt.Sprintf(a.Command.Prompt, input))
	} else {
		a.AddMessage("user", a.Command.Prompt)
	}
}

// NewAgent creates a new Agent instance
func NewAgent(id string, model *Model) *Agent {
	return &Agent{
		ID:       id,
		Model:    model,
		Messages: make([]Message, 0),
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
