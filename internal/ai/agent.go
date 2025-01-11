package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/prompt"
	"github.com/y0ug/ai-helper/pkg/mcpclient"
)

type AIConversation interface {
	LoadCommand(cmd *config.Command) error
	ApplyCommand(input string) error
	Save() error
	SendRequest() (Response, error)
	GetMessages() []Message
	AddMessage(role, content string)
}

var _ AIConversation = (*Agent)(nil) // Ensures Agent implements AIConversation

// AgentState represents the serializable state of an Agent
type AgentState struct {
	ID                string               `json:"id"`
	MCPServerCommand  string               `json:"mcp_server_command,omitempty"`
	MCPServerArgs     []string             `json:"mcp_server_args,omitempty"`
	ModelName         string               `json:"model"`
	Messages          []Message            `json:"messages"`
	Command           *config.Command      `json:"command,omitempty"`
	TemplateData      *prompt.TemplateData `json:"-"` // Skip normal JSON marshaling
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
	TotalInputTokens  int                  `json:"total_input_tokens"`
	TotalOutputTokens int                  `json:"total_output_tokens"`
	TotalCost         float64              `json:"total_cost"`
}

// Agent represents an AI conversation agent that maintains state and history
type Agent struct {
	ID                string // Unique identifier for this agent/session
	Model             *Model // The AI model being used
	Client            AIClient
	MCPClient         mcpclient.MCPClientInterface
	Messages          []Message            // Conversation history
	Command           *config.Command      // Current active command
	TemplateData      *prompt.TemplateData // Data for template processing
	CreatedAt         time.Time            // When the agent was created
	UpdatedAt         time.Time            // Last time the agent was updated
	TotalInputTokens  int                  // Total tokens used in inputs
	TotalOutputTokens int                  // Total tokens used in outputs
	TotalCost         float64              // Total cost accumulated
}

// MarshalJSON implements custom JSON marshaling for AgentState
func (s AgentState) MarshalJSON() ([]byte, error) {
	type Alias AgentState // Create alias to avoid recursion

	// Create sanitized copy of template data
	sanitizedData := *s.TemplateData
	sanitizedData.Env = make(map[string]string)

	// Copy environment vars, censoring those ending with _API_KEY
	for k, v := range s.TemplateData.Env {
		if strings.HasSuffix(strings.ToUpper(k), "_API_KEY") {
			sanitizedData.Env[k] = "********"
		} else {
			sanitizedData.Env[k] = v
		}
	}

	// Use the alias type with our sanitized data
	return json.Marshal(&struct {
		Alias
		TemplateData *prompt.TemplateData `json:"template_data"`
	}{
		Alias:        Alias(s),
		TemplateData: &sanitizedData,
	})
}

// LoadCommand loads a command configuration into the agent
func (a *Agent) LoadCommand(cmd *config.Command) error {
	a.Command = cmd

	// Initialize MCP client if MCPServers are configured
	if len(cmd.MCPServers) > 0 {
		// For now, just use the first server config
		serverName := cmd.MCPServers[0]
		if err := a.InitializeMCPClient(serverName); err != nil {
			return fmt.Errorf("failed to initialize MCP client: %w", err)
		}
	}

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
func NewAgent(id string, model *Model, client *Client) *Agent {
	now := time.Now()
	return &Agent{
		ID:           id,
		Client:       client,
		Model:        model,
		Messages:     make([]Message, 0),
		TemplateData: prompt.NewTemplateData(""),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// Save persists the agent's state to a JSON file
// InitializeMCPClient creates and initializes an MCP client for the given server name
func (a *Agent) InitializeMCPClient(serverName string) error {
	if a.Command == nil {
		return fmt.Errorf("no command loaded")
	}

	// Get server config from command configuration
	serverConfig, ok := a.Command.MCPServers[serverName]
	if !ok {
		return fmt.Errorf("MCP server %s not found in configuration", serverName)
	}

	// Create new MCP client
	client, err := mcpclient.NewMCPClient(serverConfig.Command, serverConfig.Args...)
	if err != nil {
		return fmt.Errorf("failed to create MCP client: %w", err)
	}

	// Initialize the client
	ctx := context.Background()
	if _, err := client.Initialize(ctx); err != nil {
		client.Close()
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	// Store the client
	a.MCPClient = client
	return nil
}

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
		ID:                a.ID,
		ModelName:         a.Model.Name,
		Messages:          a.Messages,
		Command:           a.Command,
		TemplateData:      a.TemplateData,
		CreatedAt:         a.CreatedAt,
		UpdatedAt:         time.Now(),
		TotalInputTokens:  a.TotalInputTokens,
		TotalOutputTokens: a.TotalOutputTokens,
		TotalCost:         a.TotalCost,
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
		ID:                state.ID,
		Model:             model,
		Messages:          state.Messages,
		Command:           state.Command,
		TemplateData:      state.TemplateData,
		CreatedAt:         state.CreatedAt,
		UpdatedAt:         state.UpdatedAt,
		TotalInputTokens:  state.TotalInputTokens,
		TotalOutputTokens: state.TotalOutputTokens,
		TotalCost:         state.TotalCost,
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

func (a *Agent) SendRequest() (Response, error) {
	resp, err := a.Client.GenerateWithMessages(a.GetMessages(), "agent_name")
	if err != nil {
		return Response{}, err
	}

	a.AddMessage("assistant", resp.Content)

	a.UpdateCosts(&resp)
	return resp, nil
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
