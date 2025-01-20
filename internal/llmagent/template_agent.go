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
	command *config.Command
	ctx     *llmcontext.RequestContext
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

	return &TemplateAgent{
		Agent:   baseAgent,
		command: command,
		ctx:     llmcontext.NewRequestContext(""),
	}, nil
}

// LoadCommand prepares the agent with the command configuration
func (ta *TemplateAgent) LoadCommand(input string) error {
	ta.ctx = llmcontext.NewRequestContext(input)
	ta.ctx.LoadEnvironment()

	// Load any files specified in the command
	if len(ta.command.Files) > 0 {
		if err := ta.ctx.LoadFiles(ta.command.Files); err != nil {
			return fmt.Errorf("failed to load command files: %w", err)
		}
	}

	// Process variables if any
	for _, v := range ta.command.Variables {
		// TODO: Implement variable processing based on type and exec
		ta.ctx.Vars[v.Name] = v.Value
	}

	return nil
}

// Execute runs the command with the prepared context
func (ta *TemplateAgent) Execute(
	ctx context.Context,
	w io.Writer,
) ([]*chat.ChatResponse, float64, error) {
	// Generate system message from template if provided
	if ta.command.System != "" {
		systemContent, err := llmcontext.Execute(ta.command.System, ta.ctx)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to execute system template: %w", err)
		}
		ta.AddMessage(chat.NewMessage("system", chat.NewTextContent(systemContent)))
	}

	// Generate user message from prompt template
	promptContent, err := llmcontext.Execute(ta.command.Prompt, ta.ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute prompt template: %w", err)
	}
	ta.AddMessage(chat.NewMessage("user", chat.NewTextContent(promptContent)))

	// Execute the chat completion
	return ta.Do(ctx, w)
}
