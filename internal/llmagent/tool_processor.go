package llmagent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/llmhaven/chat"
	"github.com/y0ug/mcpkit"
)

// go:generate go run go.uber.org/mock/mockgen@latest -destination=mock.go -package=llmagent . ToolProcessor

type ToolProcessor interface {
	GetTools() []chat.Tool
	Start(ctx context.Context, config *config.MCPServers) error
	Stop()
	HandleChoice(ctx context.Context, choice *chat.ChatChoice) ([]*chat.ChatMessage, error)
}

// MCPToolService handles the execution of tools requested by the LLM
type MCPToolService struct {
	handlers map[string]ToolHandler
	clients  map[string]mcpkit.Client
	tools    []chat.Tool
	cancel   context.CancelFunc
	logger   *slog.Logger
}

// NewToolProcessor creates a new tool processor
func NewToolProcessor(logger *slog.Logger) *MCPToolService {
	return &MCPToolService{
		handlers: make(map[string]ToolHandler),
		clients:  make(map[string]mcpkit.Client),
		tools:    make([]chat.Tool, 0),
		logger:   logger,
	}
}

// RegisterTool adds a tool handler to the processor
func (tp *MCPToolService) registerTool(name string, handler ToolHandler) {
	tp.handlers[name] = handler
}

func (tp *MCPToolService) GetTools() []chat.Tool {
	return tp.tools
}

// ProcessToolCall handles a tool call request from the LLM
func (tp *MCPToolService) processToolCall(
	ctx context.Context,
	content *chat.MessageContent,
) (*chat.ChatMessage, error) {
	if content.GetType() != string(chat.ContentTypeToolUse) {
		return nil, fmt.Errorf("invalid tool call: no tool call data")
	}

	logger := tp.logger.With("content", content.Name)
	handler, exists := tp.handlers[content.Name]
	if !exists {
		return nil, fmt.Errorf("unknown tool: %s", content.Name)
	}

	var input map[string]interface{}
	err := json.Unmarshal([]byte(content.Input), &input)
	// fmt.Println(content.InputJson)
	if err != nil {
		logger.Debug("Error unmarshalling input",
			"name", content.Name,
			"input", string(content.Input))

		return nil, fmt.Errorf("error unmarshalling input: %w", err)
	}
	logger.Debug("Tool call",
		"name", content.Name,
		"id", content.ID,
		"input", input)
	response, err := handler(ctx, input)
	if err != nil {
		logger.Error("Error executing tool",
			"error", err,
			"name", content.Name)
		return nil, fmt.Errorf("error executing tool: %w", err)
	}
	b, err := json.Marshal(response)
	if err != nil {
		logger.Error("Failed to Marshall response",
			"error", err,
			"name", content.Name)
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}
	logger.Debug("Tool result",
		"name", content.Name,
		"result", response)
	return chat.NewMessage("tool", chat.NewToolResultContent(content.ID, string(b))), nil
}

// HandleToolLoop processes tool calls in a loop until completion
func (tp *MCPToolService) HandleChoice(
	ctx context.Context,
	choice *chat.ChatChoice,
) ([]*chat.ChatMessage, error) {
	currentMessages := make([]*chat.ChatMessage, 0)

	for _, content := range choice.Content {

		if content.GetType() != string(chat.ContentTypeToolUse) {
			break
		}

		toolResult, err := tp.processToolCall(ctx, content)
		if err != nil {
			return currentMessages, fmt.Errorf("tool processing failed: %w", err)
		}

		currentMessages = append(currentMessages, toolResult)
	}

	return currentMessages, nil
}

// InitializeMCPClient
func (a *MCPToolService) Start(ctx context.Context, config *config.MCPServers) error {
	ctx, a.cancel = context.WithCancel(ctx)

	for serverName, config := range *config {
		a.logger.Debug("starting", "name", serverName)
		if _, ok := a.clients[serverName]; ok {
			a.logger.Warn("already started", "name", serverName)
			continue
		}

		// Create new MCP client
		client, err := mcpkit.NewClient(ctx, a.logger, config.Command, config.Args...)
		if err != nil {
			return fmt.Errorf("failed to create MCP client: %w", err)
		}

		if _, err := client.Initialize(ctx); err != nil {
			client.Close()
			return fmt.Errorf("failed to initialize MCP client: %w", err)
		}

		a.clients[serverName] = client

	}
	return a.setTools()
}

func (a *MCPToolService) setTools() error {
	for _, v := range a.clients {
		tools, err := mcpkit.FetchAll(context.Background(), v.ListTools)
		if err != nil {
			// a.logger.Warn("fetchTools", "name", k)
			continue
		}
		a.tools = append(a.tools, MCPClientToolToTool(tools...)...)
		for _, tool := range tools {
			a.registerTool(tool.Name, GetToolHandler(v, tool.Name))
		}
	}
	return nil
}

func (a *MCPToolService) Stop() {
	// TODO: Verify ctx implementation on both side mcpclient and Agent
	// if a.cancel != nil {
	// 	a.cancel()
	// }
	for _, client := range a.clients {
		client.Close()
	}
}
