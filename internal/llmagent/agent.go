package llmagent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/y0ug/llmhaven"
	"github.com/y0ug/llmhaven/chat"
	"github.com/y0ug/llmhaven/http/options"
	"github.com/y0ug/llmhaven/modelinfo"
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

	toolProcessor     ToolProcessor
	CreatedAt         time.Time // When the agent was created
	UpdatedAt         time.Time // Last time the agent was updated
	TotalInputTokens  int       // Total tokens used in inputs
	TotalOutputTokens int       // Total tokens used in outputs
	TotalCost         float64   // Total cost accumulated
}

func New(
	id string,
	logger *slog.Logger,
	chatParams *chat.ChatParams,
	modelInfoProvider modelinfo.Provider,
	toolProcessor ToolProcessor,
	requestOpts ...options.RequestOption,
) (*Agent, error) {
	now := time.Now()
	a := &Agent{
		ID:                id,
		logger:            logger,
		CreatedAt:         now,
		UpdatedAt:         now,
		requestOpts:       requestOpts,
		modelInfoProvider: modelInfoProvider,
		toolProcessor:     toolProcessor,
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
	modelInfo, err := modelinfo.Get(model, a.modelInfoProvider)
	a.logger.Debug("SetModel",
		"model", model,
		"provider", modelInfo.Provider,
		"info", modelInfo.Info)
	if err != nil {
		return fmt.Errorf("failed to parse model %s: %w", model, err)
	}
	provider, err := llmhaven.New(modelInfo.Provider, a.requestOpts...)
	if err != nil {
		return fmt.Errorf("failed to create provider %s / %s", modelInfo.Provider, model)
	}

	a.Client = provider
	a.ModelInfo = modelInfo
	a.chatParams.Model = a.ModelInfo.Name
	return nil
}

func (a *Agent) SaveSession() *AgentSessionState {
	model := ""
	messages := make([]*chat.ChatMessage, 0)
	if a.chatParams != nil {
		model = a.chatParams.Model
		messages = a.chatParams.Messages
	}

	return &AgentSessionState{
		ID:                a.ID,
		ModelName:         model,
		Messages:          messages,
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

// UpdateCosts updates the agent's token and cost tracking with a new response
func (a *Agent) UpdateCosts(resp ...*chat.ChatResponse) float64 {
	var cost float64
	for _, m := range resp {
		a.TotalInputTokens += m.Usage.InputTokens
		a.TotalOutputTokens += m.Usage.OutputTokens

		if a.ModelInfo == nil {
			a.logger.Warn("ModelInfo is nil, can't calculate cost")
			continue
		}
		if a.ModelInfo.Info == nil {
			a.logger.Warn("Model metadata is nil, can't calculate cost",
				"name", a.ModelInfo.Name)
			continue
		}
		//   ptrFloat := a.ModelInfo.Info.GetCacheCreationInputTokenCost()
		//   if ptrFloat != nil {
		// cost += *a.ModelInfo.Info.GetCacheCreationInputTokenCost() .OutputCostPerToken * float64(
		// 	m.Usage.OutputTokens,
		// )
		// cost += a.ModelInfo.Metadata.InputCostPerToken * float64(m.Usage.InputTokens)
	}

	a.TotalCost += cost
	return cost
}

func (a *Agent) Do(ctx context.Context, w io.Writer) ([]*chat.ChatResponse, float64, error) {
	if a.Client == nil {
		return nil, 0, fmt.Errorf("no client available")
	}

	a.chatParams.Tools = a.toolProcessor.GetTools()
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

		logger := a.logger.With("model", a.chatParams.Model)

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
		curMsgLen := len(a.GetMessages())
		// for _, choice := range msg.Choice {
		choice := msg.Choice[0]
		if a.toolProcessor != nil {
			toolResults, err := a.toolProcessor.HandleChoice(ctx, &choice)
			if err != nil {
				logger.Error("Error handling tool choice", "error", err)
			} else if len(toolResults) > 0 {
				a.AddMessage(toolResults...)
			}
		}

		// for _, content := range choice.Content {
		// }

		// No new messages, break out of loop
		if curMsgLen == len(a.GetMessages()) {
			break
		}
	}
	if w != nil {
		fmt.Fprintf(w, "\n")
	}
	return resp, nil
}

func (a *Agent) Reset() {
	a.ClearMessages()
	a.TotalInputTokens = 0
	a.TotalOutputTokens = 0
	a.TotalCost = 0
}

func (a *Agent) AddMessage(msg ...*chat.ChatMessage) {
	for _, m := range msg {
		a.logger.Debug("AddMessage", "messages", m.Content[0].String())
	}
	a.chatParams.Messages = append(a.chatParams.Messages, msg...)
}

// GetMessages returns the current message history
func (a *Agent) GetMessages() []*chat.ChatMessage {
	return a.chatParams.Messages
}

func (a *Agent) ClearMessages() {
	a.chatParams.Messages = make([]*chat.ChatMessage, 0)
}
