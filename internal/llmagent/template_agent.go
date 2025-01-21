package llmagent

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/pkg/conversation"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
	"github.com/y0ug/ai-helper/pkg/llmclient/modelinfo"
)

// TemplateAgent represents an AI agent that works with templates and commands
type TemplateAgent struct {
	*Agent
	conversation  conversation.Manager
	toolProcessor ToolProcessor
	Command       *config.Command
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

	conv := conversation.NewConversationManager(id, logger, command)

	return &TemplateAgent{
		Agent:         baseAgent,
		conversation:  conv,
		toolProcessor: toolProcessor,
	}, nil
}

func (ta *TemplateAgent) GetConversation() conversation.Manager {
	return ta.conversation
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
	// Process the turn and get messages through conversation package
	messages, err := ta.conversation.ProcessTurn(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to process turn: %w", err)
	}

	// Clear previous messages and add the new ones
	ta.ClearMessages()
	ta.AddMessage(messages...)

	// Execute chat completion

	responses, cost, err := ta.Do(ctx, w)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, cost, fmt.Errorf("request timed out after deadline: %w", err)
		}
		return nil, cost, fmt.Errorf("chat completion failed: %w", err)
	}

	// Run post-processing handlers with responses
	template := ta.conversation.GetCurrentTemplate()
	for _, handler := range template.Handlers {
		if err := handler.PostProcess(ctx, ta.conversation, responses, w); err != nil {
			return responses, cost, fmt.Errorf("post-process error: %w", err)
		}
	}

	if len(responses) == 0 {
		return nil, cost, fmt.Errorf("no response received from model")
	}

	// conversation with messages
	for _, resp := range responses {
		ta.conversation.AddMessage(resp.ToMessageParams())
	}

	// Handle state transition
	nextState, err := conversation.NewDefaultStateTransitioner().
		DetermineNextState(ta.conversation, responses)
	if err != nil {
		ta.logger.Warn("State transition error", "error", err)
	} else {
		currentState := ta.conversation.GetCurrentState()
		if nextState != currentState {
			ta.logger.Info("State transition", "from", currentState, "to", nextState)
			if err := ta.conversation.UpdateState(nextState); err != nil {
				ta.logger.Error("Failed to update state", "error", err)
			}
		}
	}

	return responses, cost, nil
}
