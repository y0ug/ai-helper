package coder

import (
	"context"
	"fmt"

	"github.com/y0ug/ai-helper/internal/ai"
	"github.com/y0ug/ai-helper/internal/coder/prompts"
)

type Coder struct {
	agent       *ai.Agent
	service     *Service
	prompts     *prompts.Manager
	initialized bool
}

func New(agent *ai.Agent) *Coder {
	return &Coder{
		agent:   agent,
		service: NewService(),
		prompts: prompts.NewManager(),
	}
}

func (c *Coder) initialize(ctx context.Context) error {
	if err := c.prompts.LoadTemplates(); err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	// Initialize with system message
	initPrompt, err := c.prompts.Execute("init", nil)
	if err != nil {
		return fmt.Errorf("failed to generate init prompt: %w", err)
	}

	msg := ai.Message{
		Role:    "system",
		Content: initPrompt,
	}
	c.agent.Messages = append(c.agent.Messages, msg)

	c.initialized = true
	return nil
}

// RequestChange handles a code change request by running it through the multi-step process
func (c *Coder) RequestChange(
	ctx context.Context,
	request string,
	files map[string]string,
) (*Response, error) {
	if !c.initialized {
		if err := c.initialize(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize coder: %w", err)
		}
	}
	return c.service.ProcessRequest(ctx, c.agent, request, files)
}
