package llmagent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/rs/zerolog"
	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/pkg/llmclient"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
	"github.com/y0ug/ai-helper/pkg/llmclient/modelinfo"
	"github.com/y0ug/ai-helper/pkg/mcpclient"
)

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
	logger            zerolog.Logger
	Model             *modelinfo.Model // The AI model being used
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
	logger zerolog.Logger,
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
	a.SetParams(chatParams)
	return a, nil
}

func (a *Agent) SetParams(chatParams *chat.ChatParams) {
	a.chatParams = chatParams
	if a.chatParams.Model != "" {
		a.SetModel(a.chatParams.Model)
	}
}

func (a *Agent) SetModel(model string) error {
	modelInfo, err := modelinfo.Parse(model, a.modelInfoProvider)
	a.logger.Debug().
		Str("model", model).
		Str("provider", modelInfo.Provider).
		Interface("metadata", modelInfo.Metadata).
		Msg("SetModel")
	if err != nil {
		return fmt.Errorf("failed to parse model %s: %w", model, err)
	}
	provider, err := llmclient.New(modelInfo.Provider, a.requestOpts...)
	if err != nil {
		return fmt.Errorf("failed to create provider %s", model)
	}

	a.Client = provider
	a.Model = modelInfo
	a.chatParams.Model = a.Model.Name
	return nil
}

// InitializeMCPClient
func (a *Agent) StartMCP(ctx context.Context) error {
	if a.mcpServerConfig == nil {
		return fmt.Errorf("no MCP servers configured")
	}

	ctx, a.mcpCancel = context.WithCancel(ctx)

	for serverName, config := range *a.mcpServerConfig {
		a.logger.Debug().Str("name", serverName).Msg("starting")
		if _, ok := a.mcpClient[serverName]; ok {
			a.logger.Warn().Str("name", serverName).Msg("already started")
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
		ModelName:         a.Model.Name,
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
			a.logger.Warn().Str("name", k).Msg("fetchTools")
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

		if a.Model.Metadata == nil {
			a.logger.Warn().
				Str("name", a.Model.Name).
				Msg("Model metadata is nil, can't calculate cost")
			continue
		}
		cost += a.Model.Metadata.OutputCostPerToken * float64(m.Usage.OutputTokens)
		cost += a.Model.Metadata.InputCostPerToken * float64(m.Usage.InputTokens)
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

		logger := a.logger.With().Str("model", a.Model.Name).Logger()
		stream, err := a.Client.Stream(ctx, *a.chatParams)
		if err != nil {
			logger.Err(err).Msg("Error streaming")
			return nil, err
		}

		eventCh := make(chan chat.EventStream)

		// llmclient.ConsumeStreamIO(ctx, stream, os.Stdout)
		go func() {
			// llmclient.ConsumeStreamIO(ctx, stream, os.Stdout)
			if err := chat.StreamChatMessageToChannel(ctx, stream, eventCh); err != nil {
				if err != context.Canceled {
					logger.Err(err).Msg("Error consuming stream")
				}
			}
		}()

		msg, err = processStream(ctx, w, eventCh)
		if err != nil {
			logger.Err(err).Msg("Error processing stream")
			return nil, nil
		}

		if msg == nil {
			logger.Err(nil).Msg("no message return")
			return resp, nil
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
					logger.Debug().Str("name", content.Name).Msg("Tool not found")
					continue
				}

				var input map[string]interface{}
				err := json.Unmarshal([]byte(content.Input), &input)
				// fmt.Println(content.InputJson)
				if err != nil {
					logger.Debug().Str("name", content.Name).Msg("Error unmarshalling input")
				}
				logger.Debug().Str("name", content.Name).
					Str("id", content.ID).
					Interface("input", input).
					Msg("Tool call")
				response, err := handler(ctx, input)
				if err != nil {
					logger.Err(err).Str("name", content.Name).Msg("Error executing tool")
					continue
				}
				b, err := json.Marshal(response)
				if err != nil {
					logger.Err(err).Str("name", content.Name).Msg("Failed to Marshall response")
				}
				toolResults = append(
					toolResults,
					chat.NewToolResultContent(content.ID, string(b)),
				)
				logger.Debug().
					Str("name", content.Name).
					Interface("result", response).
					Msg("Tool result")
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
