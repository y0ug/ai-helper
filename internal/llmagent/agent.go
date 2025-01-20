package llmagent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/pkg/llmclient"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
	"github.com/y0ug/ai-helper/pkg/llmclient/modelinfo"
	"github.com/y0ug/ai-helper/pkg/mcpclient"
)

// NOP go:generate go run go.uber.org/mock/mockgen@latest -destination=mock.go -package=llmagent .  Agenter
type AgentSessionState struct {
	ID                string              `json:"id"`
	ModelName         string              `json:"model_name"`
	Messages          []*chat.ChatMessage `json:"messages"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
	TotalInputTokens  int                 `json:"total_input_tokens"`
	TotalOutputTokens int                 `json:"total_output_tokens"`
	TotalCost         float64             `json:"total_cost"`
	// ... possibly other fields
}

// Agent represents an AI conversation agent that maintains state and history
type Agent struct {
	ID                string // Unique identifier for this agent/session
	logger            *slog.Logger
	ModelInfo         *modelinfo.Model // The AI model being used
	Client            chat.Provider
	modelInfoProvider modelinfo.Provider

	requestOpts []options.RequestOption
	chatParams  *chat.ChatParams

	mcpClient         map[string]mcpclient.MCPClientInterface
	mcpServerConfig   *config.MCPServers // List of current available MCP server configuration
	mcpCancel         context.CancelFunc
	ToolsHandler      map[string]ToolHandler // Map of tools function name to the real function
	Tools             []chat.Tool            // List of tools
	CreatedAt         time.Time              // When the agent was created
	UpdatedAt         time.Time              // Last time the agent was updated
	TotalInputTokens  int                    // Total tokens used in inputs
	TotalOutputTokens int                    // Total tokens used in outputs
	TotalCost         float64                // Total cost accumulated
}

func New(
	id string,
	logger *slog.Logger,
	chatParams *chat.ChatParams,
	modelInfoProvider modelinfo.Provider,
	mcpServersConfig *config.MCPServers,
	requestOpts ...options.RequestOption,
) (*Agent, error) {
	now := time.Now()
	a := &Agent{
		ID:                id,
		logger:            logger,
		mcpServerConfig:   mcpServersConfig,
		mcpClient:         make(map[string]mcpclient.MCPClientInterface),
		CreatedAt:         now,
		UpdatedAt:         now,
		requestOpts:       requestOpts,
		modelInfoProvider: modelInfoProvider,
	}

	if chatParams == nil {
		chatParams = chat.NewChatParams(chat.WithMaxTokens(1024))
	}
	err := a.SetParams(chatParams)
	if err != nil {
		return nil, fmt.Errorf("failed to set params: %w", err)
	}
	return a, nil
}

func (a *Agent) SetParams(chatParams *chat.ChatParams) error {
	a.chatParams = chatParams
	if a.chatParams.Model != "" {
		return a.SetModel(a.chatParams.Model)
	}
	return nil
}

func (a *Agent) SetModel(model string) error {
	modelInfo, err := modelinfo.Parse(model, a.modelInfoProvider)
	a.logger.Debug("SetModel",
		"model", model,
		"provider", modelInfo.Provider,
		"metadata", modelInfo.Metadata)
	if err != nil {
		return fmt.Errorf("failed to parse model %s: %w", model, err)
	}
	provider, err := llmclient.New(modelInfo.Provider, a.requestOpts...)
	if err != nil {
		return fmt.Errorf("failed to create provider %s / %s", modelInfo.Provider, model)
	}

	if modelInfo == nil {
		return fmt.Errorf("modelInfo is nil")
	}
	a.Client = provider
	a.ModelInfo = modelInfo
	a.chatParams.Model = a.ModelInfo.Name
	return nil
}

// InitializeMCPClient
func (a *Agent) StartMCP(ctx context.Context) error {
	if a.mcpServerConfig == nil {
		return fmt.Errorf("no MCP servers configured")
	}

	ctx, a.mcpCancel = context.WithCancel(ctx)

	for serverName, config := range *a.mcpServerConfig {
		a.logger.Debug("starting", "name", serverName)
		if _, ok := a.mcpClient[serverName]; ok {
			a.logger.Warn("already started", "name", serverName)
			continue
		}

		// Create new MCP client
		client, err := mcpclient.NewMCPClient(ctx, a.logger, config.Command, config.Args...)
		if err != nil {
			return fmt.Errorf("failed to create MCP client: %w", err)
		}

		if _, err := client.Initialize(ctx); err != nil {
			client.Close()
			return fmt.Errorf("failed to initialize MCP client: %w", err)
		}

		// Store the client
		a.mcpClient[serverName] = client

	}
	return a.setTools()
}

func (a *Agent) SaveSession() *AgentSessionState {
	return &AgentSessionState{
		ID:                a.ID,
		ModelName:         a.ModelInfo.Name,
		Messages:          a.chatParams.Messages,
		CreatedAt:         a.CreatedAt,
		UpdatedAt:         a.UpdatedAt,
		TotalInputTokens:  a.TotalInputTokens,
		TotalOutputTokens: a.TotalOutputTokens,
		TotalCost:         a.TotalCost,
	}
}

func (a *Agent) LoadSession(state *AgentSessionState) error {
	a.ID = state.ID
	a.chatParams.Messages = state.Messages
	a.CreatedAt = state.CreatedAt
	a.UpdatedAt = state.UpdatedAt
	a.TotalInputTokens = state.TotalInputTokens
	a.TotalOutputTokens = state.TotalOutputTokens
	a.TotalCost = state.TotalCost

	// If there's a model name in the session, re-set the model
	if state.ModelName != "" {
		if err := a.SetModel(state.ModelName); err != nil {
			return fmt.Errorf("failed to load model from session: %w", err)
		}
	}

	return nil
}

func (a *Agent) StopMCP() {
	// TODO: Verify ctx implementation on both side mcpclient and Agent
	if a.mcpCancel != nil {
		a.mcpCancel()
	}
	for _, client := range a.mcpClient {
		client.Close()
	}
}

func (a *Agent) setTools() error {
	a.ToolsHandler = make(map[string]ToolHandler)
	a.Tools = make([]chat.Tool, 0)

	for k, v := range a.mcpClient {
		tools, err := mcpclient.FetchAll(context.Background(), v.ListTools)
		if err != nil {
			a.logger.Warn("fetchTools", "name", k)
			continue
		}
		a.Tools = append(a.Tools, MCPClientToolToTool(tools...)...)
		for _, tool := range tools {
			a.ToolsHandler[tool.Name] = GetToolHandler(v, tool.Name)
		}
	}
	return nil
}

// UpdateCosts updates the agent's token and cost tracking with a new response
func (a *Agent) UpdateCosts(resp ...*chat.ChatResponse) float64 {
	var cost float64
	for _, m := range resp {
		a.TotalInputTokens += m.Usage.InputTokens
		a.TotalOutputTokens += m.Usage.OutputTokens

		if a.ModelInfo.Metadata == nil {
			a.logger.Warn("Model metadata is nil, can't calculate cost",
				"name", a.ModelInfo.Name)
			continue
		}
		cost += a.ModelInfo.Metadata.OutputCostPerToken * float64(m.Usage.OutputTokens)
		cost += a.ModelInfo.Metadata.InputCostPerToken * float64(m.Usage.InputTokens)
	}

	a.TotalCost += cost
	return cost
}

func (a *Agent) Do(ctx context.Context, w io.Writer) ([]*chat.ChatResponse, float64, error) {
	a.chatParams.Tools = a.Tools
	resp, err := a.process(ctx, w)
	cost := a.UpdateCosts(resp...)
	return resp, cost, err
}

func processStream(
	ctx context.Context,
	w io.Writer,
	ch <-chan chat.EventStream,
) (*chat.ChatResponse, error) {
	var cm *chat.ChatResponse
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case set, ok := <-ch:
			if !ok {
				return cm, nil
			}
			if set.Type == "text_delta" {
				if w != nil {
					fmt.Fprintf(w, "%v", set.Delta)
				}
			}
			if set.Type == "message_stop" {
				cm = set.Message
			}
		}
	}
}

func (a *Agent) process(
	ctx context.Context,
	w io.Writer,
) ([]*chat.ChatResponse, error) {
	resp := make([]*chat.ChatResponse, 0)
	var msg *chat.ChatResponse
	for {

		logger := a.logger.With("test", "") // slog.With("model", a.Model.Name)

		stream, err := a.Client.Stream(ctx, *a.chatParams)
		if err != nil {
			logger.Error("Error streaming", "error", err)
			return nil, err
		}

		eventCh := make(chan chat.EventStream)

		// llmclient.ConsumeStreamIO(ctx, stream, os.Stdout)
		go func() {
			// llmclient.ConsumeStreamIO(ctx, stream, os.Stdout)
			if err := chat.StreamChatMessageToChannel(ctx, stream, eventCh); err != nil {
				if err != context.Canceled {
					logger.Error("Error consuming stream", "error", err)
				}
			}
		}()

		msg, err = processStream(ctx, w, eventCh)
		if err != nil {
			logger.Error("Error processing stream", "error", err)
			return nil, nil
		}

		if msg == nil {
			logger.Error("no message return")
			return nil, fmt.Errorf("no message returned from LLM")
		}
		resp = append(resp, msg)

		a.AddMessage(msg.ToMessageParams())
		toolResults := make([]*chat.MessageContent, 0)
		// for _, choice := range msg.Choice {
		choice := msg.Choice[0]
		for _, content := range choice.Content {
			if content.Type == "tool_use" {
				handler, ok := a.ToolsHandler[content.Name]
				if !ok {
					logger.Debug("Tool not found", "name", content.Name)
					continue
				}

				var input map[string]interface{}
				err := json.Unmarshal([]byte(content.Input), &input)
				// fmt.Println(content.InputJson)
				if err != nil {
					logger.Debug("Error unmarshalling input",
						"name", content.Name,
						"input", string(content.Input))
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
					continue
				}
				b, err := json.Marshal(response)
				if err != nil {
					logger.Error("Failed to Marshall response",
						"error", err,
						"name", content.Name)
				}
				toolResults = append(
					toolResults,
					chat.NewToolResultContent(content.ID, string(b)),
				)
				logger.Debug("Tool result",
					"name", content.Name,
					"result", response)
			}
		}
		if len(toolResults) == 0 {
			break
		}
		a.AddMessage(chat.NewMessage("tool", toolResults...))
	}
	if w != nil {
		fmt.Fprintf(w, "\n")
	}
	return resp, nil
}

func (a *Agent) Reset() {
	a.chatParams.Messages = make([]*chat.ChatMessage, 0)
	a.TotalInputTokens = 0
	a.TotalOutputTokens = 0
	a.TotalCost = 0
}

func (a *Agent) AddMessage(msg ...*chat.ChatMessage) {
	a.chatParams.Messages = append(a.chatParams.Messages, msg...)
}

// GetMessages returns the current message history
func (a *Agent) GetMessages() []*chat.ChatMessage {
	return a.chatParams.Messages
}
