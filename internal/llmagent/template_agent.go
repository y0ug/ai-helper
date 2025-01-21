package llmagent

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
	"github.com/y0ug/ai-helper/pkg/llmclient/modelinfo"
)

// TemplateAgent represents an AI agent that works with templates and commands
type TemplateAgent struct {
	*Agent
	ConversationManager *ConversationManager
	StateTransitioner   StateTransitioner
	Command             *config.Command
}

// NewTemplateAgent creates a new template-based agent
func NewTemplateAgent(
	id string,
	logger *slog.Logger,
	command *config.Command,
	chatParams *chat.ChatParams,
	modelInfoProvider modelinfo.Provider,
	toolProcessor ToolProcessor,
	requestOpts ...options.RequestOption,
) (*TemplateAgent, error) {
	baseAgent, err := New(
		id,
		logger,
		chatParams,
		modelInfoProvider,
		toolProcessor,
		requestOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create base agent: %w", err)
	}

	conversationManager := NewConversationManager(id, logger)
	if err := conversationManager.LoadCommand(command); err != nil {
		return nil, fmt.Errorf("failed to load command: %w", err)
	}

	return &TemplateAgent{
		Agent:               baseAgent,
		ConversationManager: conversationManager,
		StateTransitioner:   NewDefaultStateTransitioner(),
	}, nil
}

func (ta *TemplateAgent) GetModelName() string {
	if ta.chatParams != nil {
		return ta.chatParams.Model
	}
	return ""
}

// Execute runs the command with the prepared context
func (ta *TemplateAgent) Execute(
	ctx context.Context,
	w io.Writer,
) ([]*chat.ChatResponse, float64, error) {
	// Process the turn and get messages through conversation manager
	turn, messages, err := ta.ConversationManager.ProcessTurn(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to process turn: %w", err)
	}

	// Clear previous messages and add the new ones
	ta.ClearMessages()
	for _, msg := range messages {
		ta.AddMessage(msg)
	}

	// Execute chat completion

	responses, cost, err := ta.Do(ctx, w)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, cost, fmt.Errorf("request timed out after deadline: %w", err)
		}
		return nil, cost, fmt.Errorf("chat completion failed: %w", err)
	}

	// Run post-processing handlers with responses
	for _, handler := range ta.ConversationManager.GetCurrentTemplate().Handlers {
		if err := handler.PostProcess(ctx, ta.ConversationManager, responses, w); err != nil {
			return responses, cost, fmt.Errorf("post-process error: %w", err)
		}
	}

	if len(responses) == 0 {
		return nil, cost, fmt.Errorf("no response received from model")
	}

	// Update turn with messages
	turn.Messages = ta.GetMessages()

	// Handle state transition
	if ta.StateTransitioner != nil {
		nextState, err := ta.StateTransitioner.DetermineNextState(ta.ConversationManager, responses)
		if err != nil {
			ta.logger.Warn("State transition error", "error", err)
		} else if nextState != ta.ConversationManager.CurrentState {
			ta.logger.Info("State transition", "from", ta.ConversationManager.CurrentState, "to", nextState)
			ta.ConversationManager.CurrentState = nextState
		}
	}

	return responses, cost, nil
}
