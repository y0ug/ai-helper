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
	"github.com/y0ug/ai-helper/pkg/llmclient"
	"github.com/y0ug/ai-helper/pkg/llmclient/requestoption"
	"github.com/y0ug/ai-helper/pkg/llmclient/types"
	"github.com/y0ug/ai-helper/pkg/mcpclient"
)

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

func MCPClientToolToTool(tools ...mcpclient.Tool) []types.Tool {
	result := make([]types.Tool, 0)
	for _, tool := range tools {
		result = append(result, types.Tool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}
	return result
}

// AgentState represents the serializable state of an Agent
type AgentState struct {
	ID                string               `json:"id"`
	ModelName         string               `json:"model"`
	Messages          []*types.ChatMessage `json:"messages"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
	TotalInputTokens  int                  `json:"total_input_tokens"`
	TotalOutputTokens int                  `json:"total_output_tokens"`
	TotalCost         float64              `json:"total_cost"`
}

// Agent represents an AI conversation agent that maintains state and history
type Agent struct {
	ID          string           // Unique identifier for this agent/session
	Model       *llmclient.Model // The AI model being used
	Client      llmclient.LLMProvider
	requestOpts []requestoption.RequestOption

	MCPClient        map[string]mcpclient.MCPClientInterface
	MCPServersConfig *config.MCPServers // List of current available MCP server configuration

	ToolsHandler      map[string]ToolHandler // Map of tools function name to the real function
	Tools             []types.Tool           // List of tools
	Messages          []*types.ChatMessage   // Conversation history
	CreatedAt         time.Time              // When the agent was created
	UpdatedAt         time.Time              // Last time the agent was updated
	TotalInputTokens  int                    // Total tokens used in inputs
	TotalOutputTokens int                    // Total tokens used in outputs
	TotalCost         float64                // Total cost accumulated
}

func NewAgent(
	id string,
	mcpServersConfig config.MCPServers,
) *Agent {
	now := time.Now()
	return &Agent{
		ID:               id,
		MCPServersConfig: &mcpServersConfig,
		MCPClient:        make(map[string]mcpclient.MCPClientInterface),
		Messages:         make([]*types.ChatMessage, 0),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func (a *Agent) SetModel(model string, requestOpts ...requestoption.RequestOption) {
	modelInfoProvider, _ := llmclient.NewModelInfoProvider("")
	provider, modelInfo := llmclient.NewProviderByModel(model, modelInfoProvider, requestOpts...)
	if provider != nil {
		a.Client = provider
		a.Model = modelInfo
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
	a.ToolsHandler = make(map[string]ToolHandler)
	a.Tools = make([]types.Tool, 0)

	for k, v := range a.MCPClient {
		tools, err := mcpclient.FetchAll(context.Background(), v.ListTools)
		if err != nil {
			fmt.Printf("fetchTools error %s:%v", k, err)
			continue
		}
		a.Tools = append(a.Tools, MCPClientToolToTool(tools...)...)
		for _, tool := range tools {
			a.ToolsHandler[tool.Name] = GetToolHandler(v, tool.Name)
		}
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
		CreatedAt:         state.CreatedAt,
		UpdatedAt:         state.UpdatedAt,
		TotalInputTokens:  state.TotalInputTokens,
		TotalOutputTokens: state.TotalOutputTokens,
		TotalCost:         state.TotalCost,
	}

	return agent, nil
}

// UpdateCosts updates the agent's token and cost tracking with a new response
func (a *Agent) UpdateCosts(response types.ChatResponse) {
	a.TotalInputTokens += response.Usage.InputTokens
	a.TotalOutputTokens += response.Usage.OutputTokens
	// if response.Cost != nil {
	// 	a.TotalCost += *response.Cost
	// }
}

func (a *Agent) SendRequest(ctx context.Context) ([]types.ChatResponse, error) {
	params := llmclient.NewChatParams(
		llmclient.WithModel(a.Model.Name),
		llmclient.WithMaxTokens(1024),
		llmclient.WithTemperature(0),
		llmclient.WithMessages(
			llmclient.NewUserMessage(

				// Can you write an Hello World in C?
				"What the weather at Paris ?",
				// "Write a 500 word essai about Golang and put a some code block in the middle",
			),
		),
		llmclient.WithTools(a.Tools...),
	)

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

func (a *Agent) AddMessage(msg ...*types.ChatMessage) {
	a.Messages = append(a.Messages, msg...)
}

// GetMessages returns the current message history
func (a *Agent) GetMessages() []*types.ChatMessage {
	return a.Messages
}
