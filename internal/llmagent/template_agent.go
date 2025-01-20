package llmagent

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/llmcontext"
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
	reqctx              *llmcontext.RequestContext
}

// NewTemplateAgent creates a new template-based agent
func NewTemplateAgent(
	id string,
	logger *slog.Logger,
	command *config.Command,
	chatParams *chat.ChatParams,
	modelInfoProvider modelinfo.Provider,
	mcpServersConfig *config.MCPServers,
	requestOpts ...options.RequestOption,
) (*TemplateAgent, error) {
	baseAgent, err := New(
		id,
		logger,
		chatParams,
		modelInfoProvider,
		mcpServersConfig,
		requestOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create base agent: %w", err)
	}

	conversationManager := NewConversationManager(id)
	if err := conversationManager.LoadCommand(command); err != nil {
		return nil, fmt.Errorf("failed to load command: %w", err)
	}

	return &TemplateAgent{
		Agent:               baseAgent,
		ConversationManager: conversationManager,
		Command:             command,
		reqctx:              llmcontext.NewRequestContext(command),
		StateTransitioner:   NewDefaultStateTransitioner(),
	}, nil
}


func (ta *TemplateAgent) LoadArgs(args map[string]string) error {
	return ta.reqctx.Process(args)
}

func (ta *TemplateAgent) LoadFiles(filePath ...string) error {
	if err := ta.reqctx.LoadFiles(filePath...); err != nil {
		return err
	}

	// Get current template and update its prompt with file contents
	// currentTemplate := ta.ConversationManager.GetCurrentTemplate()
	// if currentTemplate != nil {
	// 	// Append file contents to the current prompt
	// 	currentTemplate.UserPrompt = fmt.Sprintf(
	// 		"%s\n\nFiles that have been added to the chat:\n{{.Files}}",
	// 		currentTemplate.UserPrompt,
	// 	)
	// }
	return nil
}

func (ta *TemplateAgent) SetInput(input string) error {
	if ta.ConversationManager.IsInputNeeded() {
		ta.logger.Debug("adding new input", "input", input)
		ta.ConversationManager.Variables["Input"] = input
		ta.reqctx.Vars["Input"] = input
	}
	return nil
}

// Execute runs the command with the prepared context
func (ta *TemplateAgent) Execute(
	ctx context.Context,
	w io.Writer,
) ([]*chat.ChatResponse, float64, error) {
	// Process the turn and get messages through conversation manager
	turn, messages, err := ta.ConversationManager.ProcessTurn(ctx, ta.reqctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to process turn: %w", err)
	}

	// Clear previous messages and add the new ones
	ta.ClearMessages()
	for _, msg := range messages {
		ta.AddMessage(msg)
	}

	// Execute the chat completion with timeout handling
	responses, cost, err := ta.Do(ctx, w)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, cost, fmt.Errorf("request timed out after deadline: %w", err)
		}
		return nil, cost, fmt.Errorf("chat completion failed: %w", err)
	}

	if len(responses) == 0 {
		return nil, cost, fmt.Errorf("no response received from model")
	}

	// Update turn with messages
	turn.Messages = ta.GetMessages()

	// Handle state transition
	if ta.StateTransitioner != nil {
		nextState, err := ta.StateTransitioner.DetermineNextState(turn, responses)
		if err != nil {
			ta.logger.Warn("State transition error", "error", err)
		} else if nextState != ta.ConversationManager.CurrentState {
			ta.logger.Info("State transition", "from", ta.ConversationManager.CurrentState, "to", nextState)
			ta.ConversationManager.CurrentState = nextState
		}
	}

	return responses, cost, nil
}
