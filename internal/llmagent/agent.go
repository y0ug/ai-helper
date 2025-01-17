package llmagent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/pkg/llmclient"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
	"github.com/y0ug/ai-helper/pkg/llmclient/modelinfo"
	"github.com/y0ug/ai-helper/pkg/mcpclient"
)

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
	Messages          []*chat.ChatMessage    // Conversation history
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
	cachePath string,
	mcpServersConfig *config.MCPServers,
	requestOpts ...options.RequestOption,
) (*Agent, error) {
	now := time.Now()
	a := &Agent{
		ID:              id,
		logger:          logger,
		mcpServerConfig: mcpServersConfig,
		mcpClient:       make(map[string]mcpclient.MCPClientInterface),
		Messages:        make([]*chat.ChatMessage, 0),
		CreatedAt:       now,
		UpdatedAt:       now,
		requestOpts:     requestOpts,
		chatParams:      chatParams,
	}

	var err error
	a.modelInfoProvider, err = modelinfo.New(filepath.Join(cachePath, "modelinfo.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to create model info provider: %w", err)
	}

	if a.chatParams.Model != "" {
		err := a.SetModel(a.chatParams.Model)
		if err != nil {
			return nil, fmt.Errorf("failed to set model: %w", err)
		}
	}
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
func (a *Agent) UpdateCosts(response chat.ChatResponse) {
	a.TotalInputTokens += response.Usage.InputTokens
	a.TotalOutputTokens += response.Usage.OutputTokens
	// if response.Cost != nil {
	// 	a.TotalCost += *response.Cost
	// }
}

func (a *Agent) Do(ctx context.Context, w io.Writer) ([]*chat.ChatResponse, error) {
	a.chatParams.Messages = a.Messages
	a.chatParams.Tools = a.Tools
	resp, err := a.process(ctx, w)
	var cost float64
	for _, m := range resp {
		a.TotalInputTokens += m.Usage.InputTokens
		a.TotalOutputTokens += m.Usage.OutputTokens

		cost += a.Model.Metadata.OutputCostPerToken * float64(m.Usage.OutputTokens)
		cost += a.Model.Metadata.InputCostPerToken * float64(m.Usage.InputTokens)
	}
	a.logger.Info().Msgf("Total cost: %f\n", cost)
	return resp, err
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
				fmt.Fprintf(w, "%v", set.Delta)
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

		a.chatParams.Messages = append(a.chatParams.Messages, msg.ToMessageParams())
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
					panic(err)
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
		a.chatParams.Messages = append(
			a.chatParams.Messages,
			chat.NewMessage("user", toolResults...),
		)
	}
	fmt.Fprintf(w, "\n")
	return resp, nil
}

func (a *Agent) Reset() {
	a.Messages = make([]*chat.ChatMessage, 0)
	a.TotalInputTokens = 0
	a.TotalOutputTokens = 0
	a.TotalCost = 0
}

func (a *Agent) AddMessage(msg ...*chat.ChatMessage) {
	a.Messages = append(a.Messages, msg...)
}

// GetMessages returns the current message history
func (a *Agent) GetMessages() []*chat.ChatMessage {
	return a.Messages
}
