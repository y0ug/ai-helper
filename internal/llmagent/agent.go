package llmagent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/pkg/highlighter"
	"github.com/y0ug/ai-helper/pkg/llmclient"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
	"github.com/y0ug/ai-helper/pkg/llmclient/modelinfo"
	"github.com/y0ug/ai-helper/pkg/mcpclient"
)

// Agent represents an AI conversation agent that maintains state and history
type Agent struct {
	ID                string           // Unique identifier for this agent/session
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

func NewAgent(
	id string,
	chatParams *chat.ChatParams,
	cachePath string,
	mcpServersConfig *config.MCPServers,
	requestOpts ...options.RequestOption,
) (*Agent, error) {
	now := time.Now()
	a := &Agent{
		ID:              id,
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
	return a, nil
}

func (a *Agent) SetParams(chatParams *chat.ChatParams) {
	a.chatParams = chatParams
	if a.chatParams.Model != "" {
		a.SetModel(a.chatParams.Model)
	}
}

func (a *Agent) SetModel(model string) error {
	provider, modelInfo := llmclient.New(model, a.modelInfoProvider, a.requestOpts...)
	if provider != nil {
		return fmt.Errorf("failed to create provider")
	}

	a.Client = provider
	a.Model = modelInfo
	a.chatParams.Model = a.Model.Name
	return nil
}

// InitializeMCPClient
func (a *Agent) startMCP(ctx context.Context) error {
	if a.mcpServerConfig == nil {
		return fmt.Errorf("no MCP servers configured")
	}

	ctx, a.mcpCancel = context.WithCancel(ctx)

	for serverName, config := range *a.mcpServerConfig {
		fmt.Printf("serverName %s", serverName)
		if _, ok := a.mcpClient[serverName]; ok {
			log.Printf("MCP client '%s' already initialized", serverName)
		}

		// Create new MCP client
		client, err := mcpclient.NewMCPClient(ctx, config.Command, config.Args...)
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

// UpdateCosts updates the agent's token and cost tracking with a new response
func (a *Agent) UpdateCosts(response chat.ChatResponse) {
	a.TotalInputTokens += response.Usage.InputTokens
	a.TotalOutputTokens += response.Usage.OutputTokens
	// if response.Cost != nil {
	// 	a.TotalCost += *response.Cost
	// }
}

func (a *Agent) Do(ctx context.Context, w io.Writer) (*chat.ChatResponse, error) {
	a.chatParams.Messages = a.Messages
	resp, err := a.process(ctx, w)
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
) (*chat.ChatResponse, error) {
	var msg *chat.ChatResponse
	for {

		stream, err := a.Client.Stream(ctx, *a.chatParams)
		if err != nil {
			log.Printf("Error streaming: %v", err)
			return nil, err
		}

		eventCh := make(chan chat.EventStream)

		// llmclient.ConsumeStreamIO(ctx, stream, os.Stdout)
		go func() {
			// llmclient.ConsumeStreamIO(ctx, stream, os.Stdout)
			if err := chat.StreamChatMessageToChannel(ctx, stream, eventCh); err != nil {
				if err != context.Canceled {
					log.Printf("Error consuming stream: %v", err)
				}
			}
		}()

		h := highlighter.NewHighlighter(os.Stdout)
		msg, err = processStream(ctx, h, eventCh)
		if err != nil {
			log.Printf("Error processing stream: %v", err)
			return nil, nil
		}

		if msg == nil {
			log.Printf("No message returned")
			return nil, nil
		}
		fmt.Printf("\nUsage: %d %d\n", msg.Usage.InputTokens, msg.Usage.OutputTokens)

		a.chatParams.Messages = append(a.chatParams.Messages, msg.ToMessageParams())
		toolResults := make([]*chat.MessageContent, 0)
		// for _, choice := range msg.Choice {
		choice := msg.Choice[0]
		for _, content := range choice.Content {
			if content.Type == "tool_use" {
				handler, ok := a.ToolsHandler[content.Name]
				if !ok {
					log.Printf("Tool %s not found", content.Name)
					continue
				}
				log.Printf(
					"%s execution: %s with \"%s\"",
					content.ID,
					content.Name,
					string(content.Input),
				)

				var input map[string]interface{}
				err := json.Unmarshal([]byte(content.Input), &input)
				// fmt.Println(content.InputJson)
				if err != nil {
					log.Printf("Error unmarshalling input: %v", err)
				}
				response, err := handler(ctx, input)
				if err != nil {
					log.Printf("Error executing tool: %v", err)
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
				log.Printf("Tool result: %s", string(b))
			}
		}
		// }
		if len(toolResults) == 0 {
			break
		}

		// if params.N != nil {
		// 	*params.N = 1
		// }

		a.chatParams.Messages = append(
			a.chatParams.Messages,
			chat.NewMessage("user", toolResults...),
		)
	}
	return msg, nil
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
