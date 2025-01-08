package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/prompt"
)

// AgentState represents the serializable state of an Agent
type AgentState struct {
	ID               string               `json:"id"`
	ModelName        string               `json:"model"`
	Messages         []Message            `json:"messages"`
	Command          *config.Command      `json:"command,omitempty"`
	TemplateData     *prompt.TemplateData `json:"template_data"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
	TotalInputTokens  int                 `json:"total_input_tokens"`
	TotalOutputTokens int                 `json:"total_output_tokens"`
	TotalCost        float64              `json:"total_cost"`
}

// Agent represents an AI conversation agent that maintains state and history
type Agent struct {
	ID               string               // Unique identifier for this agent/session
	Model            *Model               // The AI model being used
	Messages         []Message            // Conversation history
	Command          *config.Command      // Current active command
	TemplateData     *prompt.TemplateData // Data for template processing
	CreatedAt        time.Time            // When the agent was created
	UpdatedAt        time.Time            // Last time the agent was updated
	TotalInputTokens  int                 // Total tokens used in inputs
	TotalOutputTokens int                 // Total tokens used in outputs
	TotalCost        float64              // Total cost accumulated
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
	now := time.Now()
	return &Agent{
		ID:           id,
		Model:        model,
		Messages:     make([]Message, 0),
		TemplateData: prompt.NewTemplateData(""),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// Save persists the agent's state to a JSON file
func (a *Agent) Save() error {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("failed to get cache directory: %w", err)
	}

	agentDir := filepath.Join(cacheDir, "ai-helper", "agents")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		return fmt.Errorf("failed to create agent directory: %w", err)
	}

	state := AgentState{
		ID:               a.ID,
		ModelName:        a.Model.String(),
		Messages:         a.Messages,
		Command:          a.Command,
		TemplateData:     a.TemplateData,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        time.Now(),
		TotalInputTokens:  a.TotalInputTokens,
		TotalOutputTokens: a.TotalOutputTokens,
		TotalCost:        a.TotalCost,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agent state: %w", err)
	}

	filename := filepath.Join(agentDir, fmt.Sprintf("%s.json", a.ID))
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write agent state: %w", err)
	}

	return nil
}

// Load restores the agent's state from a JSON file
func LoadAgent(id string, model *Model) (*Agent, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}

	filename := filepath.Join(cacheDir, "ai-helper", "agents", fmt.Sprintf("%s.json", id))
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent state: %w", err)
	}

	var state AgentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent state: %w", err)
	}

	agent := &Agent{
		ID:               state.ID,
		Model:            model,
		Messages:         state.Messages,
		Command:          state.Command,
		TemplateData:     state.TemplateData,
		CreatedAt:        state.CreatedAt,
		UpdatedAt:        state.UpdatedAt,
		TotalInputTokens:  state.TotalInputTokens,
		TotalOutputTokens: state.TotalOutputTokens,
		TotalCost:        state.TotalCost,
	}

	return agent, nil
}

// UpdateCosts updates the agent's token and cost tracking with a new response
func (a *Agent) UpdateCosts(response *Response) {
	a.TotalInputTokens += response.InputTokens
	a.TotalOutputTokens += response.OutputTokens
	if response.Cost != nil {
		a.TotalCost += *response.Cost
	}
}

// ListAgents returns a list of all saved agent IDs
func ListAgents() ([]string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}

	agentDir := filepath.Join(cacheDir, "ai-helper", "agents")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create agent directory: %w", err)
	}

	files, err := os.ReadDir(agentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent directory: %w", err)
	}

	var agents []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			agents = append(agents, strings.TrimSuffix(file.Name(), ".json"))
		}
	}

	return agents, nil
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
