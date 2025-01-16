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
	"github.com/y0ug/ai-helper/pkg/llmclient"
	"github.com/y0ug/ai-helper/pkg/llmclient/common"
	"github.com/y0ug/ai-helper/pkg/mcpclient"
)

type AIConversation interface {
	LoadCommand(cmd *config.Command) error
	ApplyCommand(input string) error
	Save() error
	SendRequest() ([]*common.ChatMessageParams, []common.ChatMessage, error)
	GetMessages() []*common.ChatMessageParams
	AddMessage(role, content string)
}

var _ AIConversation = (*Agent)(nil) // Ensures Agent implements AIConversation

type ToolHandler func(ctx context.Context, input map[string]interface{}) ([]interface{}, error)

func GetToolHandler(c mcpclient.MCPClientInterface, name string) ToolHandler {
	return func(ctx context.Context, input map[string]interface{}) ([]interface{}, error) {
		result, err := c.CallTool(ctx, name, input)
		if err != nil {
			return nil, err
		}
		return result.Content, nil
	}
}

// AgentState represents the serializable state of an Agent
type AgentState struct {
	ID                string                     `json:"id"`
	ModelName         string                     `json:"model"`
	Messages          []common.ChatMessageParams `json:"messages"`
	Command           *config.Command            `json:"command,omitempty"`
	TemplateData      *prompt.TemplateData       `json:"-"` // Skip normal JSON marshaling
	CreatedAt         time.Time                  `json:"created_at"`
	UpdatedAt         time.Time                  `json:"updated_at"`
	TotalInputTokens  int                        `json:"total_input_tokens"`
	TotalOutputTokens int                        `json:"total_output_tokens"`
	TotalCost         float64                    `json:"total_cost"`
}

// Agent represents an AI conversation agent that maintains state and history
type Agent struct {
	ID     string           // Unique identifier for this agent/session
	Model  *llmclient.Model // The AI model being used
	Client llmclient.LLMProvider

	MCPClient        map[string]mcpclient.MCPClientInterface
	MCPServersConfig *config.MCPServers // List of current available MCP server configuration

	ToolsHandler      map[string]ToolHandler      // Map of tools function name to the real function
	Tools             []common.Tool               // List of tools
	Messages          []*common.ChatMessageParams // Conversation history
	Command           *config.Command             // Current active command
	TemplateData      *prompt.TemplateData        // Data for template processing
	CreatedAt         time.Time                   // When the agent was created
	UpdatedAt         time.Time                   // Last time the agent was updated
	TotalInputTokens  int                         // Total tokens used in inputs
	TotalOutputTokens int                         // Total tokens used in outputs
	TotalCost         float64                     // Total cost accumulated
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
	for _, v := range cmd.MCPServers {
		err := a.InitializeMCPClient(v)
		if err != nil {
			fmt.Printf("failed to start mcp client %s %v", v, err)
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

	a.setTools()
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
func NewAgent(
	id string,
	model *llmclient.Model,
	client llmclient.LLMProvider,
	mcpServersConfig config.MCPServers,
) *Agent {
	now := time.Now()
	return &Agent{
		ID:               id,
		Client:           client,
		Model:            model,
		MCPServersConfig: &mcpServersConfig,
		MCPClient:        make(map[string]mcpclient.MCPClientInterface),
		Messages:         make([]*common.ChatMessageParams, 0),
		TemplateData:     prompt.NewTemplateData(""),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// Save persists the agent's state to a JSON file
// InitializeMCPClient creates and initializes an MCP client for the given server name
func (a *Agent) InitializeMCPClient(serverName string) error {
	fmt.Printf("serverName %s", serverName)
	if a.MCPServersConfig == nil {
		return fmt.Errorf("no MCP servers configured")
	}
	config, ok := (*a.MCPServersConfig)[serverName]
	if !ok {
		return fmt.Errorf("MCP server '%s' not found in configuration", serverName)
	}
	if _, ok := a.MCPClient[serverName]; ok {
		return fmt.Errorf("MCP client '%s' already initialized", serverName)
	}

	// Create new MCP client
	client, err := mcpclient.NewMCPClient(config.Command, config.Args...)
	if err != nil {
		return fmt.Errorf("failed to create MCP client: %w", err)
	}

	// Initialize the client
	// TODO: should do proper ctx tracking
	// to be able to stop properly
	ctx := context.Background()
	if _, err := client.Initialize(ctx); err != nil {
		client.Close()
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	// Store the client
	a.MCPClient[serverName] = client
	return nil
}

func (a *Agent) setTools() error {
	a.Tools = make(map[string]llmclient.ToolHandler)
	aiTools := make([]llmclient.AITools, 0)
	for k, v := range a.MCPClient {
		tools, err := mcpclient.FetchAll(context.Background(), v.ListTools)
		if err != nil {
			fmt.Printf("fetchTools error %s:%v", k, err)
			continue
		}
		aiTools = append(aiTools, llmclient.ToAITools(tools)...)
		for _, tool := range tools {
			a.Tools[tool.Name] = llmclient.GetToolHandler(v, tool.Name)
		}
	}
	if a.Client != nil {
		aiTools := aiTools
		a.Client.SetTools(aiTools)
	}
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
func LoadAgent(id string, model *llmclient.Model) (*Agent, error) {
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
func (a *Agent) UpdateCosts(response llmclient.AIResponse) {
	a.TotalInputTokens += response.GetUsage().GetInputTokens()
	a.TotalOutputTokens += response.GetUsage().GetOutputTokens()
	// if response.Cost != nil {
	// 	a.TotalCost += *response.Cost
	// }
}

func (a *Agent) SendRequest() ([]llmclient.AIMessage, []llmclient.AIResponse, error) {
	message, response, err := a.Client.ProcessMessages(a.Tools, a.Messages...)
	return message, response, err
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
	a.Messages = append(
		a.Messages,
		llmclient.NewBaseMessage(role, llmclient.NewTextContent(content)),
	)
}

func (a *Agent) AddMessageM(msg llmclient.AIMessage) {
	a.Messages = append(a.Messages, msg)
}

// GetMessages returns the current message history
func (a *Agent) GetMessages() []llmclient.AIMessage {
	return a.Messages
}

// AddSystemMessage adds a system message to the start of the conversation
func (a *Agent) AddSystemMessage(content string) {
	// TODO: Implements with AIMesssage
	// // If first message is already a system message, replace it
	// if len(a.Messages) > 0 && a.Messages[0].Role == "system" {
	// 	a.Messages[0].Content = content
	// 	return
	// }
	//
	// // Insert system message at the beginning
	// a.Messages = append([]Message{{Role: "system", Content: content}}, a.Messages...)
}
