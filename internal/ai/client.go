package ai

import (
	"fmt"
	"os"
	
	"github.com/y0ug/ai-helper/internal/cost"
	"github.com/y0ug/ai-helper/internal/stats"
)

const (
	EnvAIModel          = "AI_MODEL"
	EnvAnthropicAPIKey  = "ANTHROPIC_API_KEY"
	EnvOpenAIAPIKey     = "OPENAI_API_KEY"
	EnvOpenRouterAPIKey = "OPENROUTER_API_KEY"
	EnvGeminiAPIKey     = "GEMINI_API_KEY"
)

// Client handles AI model interactions
type Client struct {
	provider Provider
	model    *Model
	tracker  *cost.Tracker
}

// NewClient creates a new AI client using environment variables
func NewClient() (*Client, error) {
	modelStr := os.Getenv(EnvAIModel)
	if modelStr == "" {
		return nil, fmt.Errorf("AI_MODEL environment variable not set")
	}

	model, err := ParseModel(modelStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse model: %w", err)
	}

	var apiKey string
	switch model.Provider {
	case "anthropic":
		apiKey = os.Getenv(EnvAnthropicAPIKey)
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
		}
	case "openai":
		apiKey = os.Getenv(EnvOpenAIAPIKey)
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
		}
	case "openrouter":
		apiKey = os.Getenv(EnvOpenRouterAPIKey)
		if apiKey == "" {
			return nil, fmt.Errorf("OPENROUTER_API_KEY environment variable not set")
		}
	case "gemini":
		apiKey = os.Getenv(EnvGeminiAPIKey)
		if apiKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
		}
	default:
		return nil, fmt.Errorf("unsupported provider: %s", model.Provider)
	}

	provider, err := NewProvider(model, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	tracker, err := cost.NewTracker()
	if err != nil {
		return nil, fmt.Errorf("failed to create cost tracker: %w", err)
	}

	return &Client{
		provider: provider,
		model:    model,
		tracker:  tracker,
	}, nil
}

// Generate sends a prompt to the AI model and returns the response
func (c *Client) Generate(prompt string, system string, command string) (Response, error) {
	messages := []Message{}
	if system != "" {
		messages = append(messages, Message{Role: "system", Content: system})
	}
	messages = append(messages, Message{Role: "user", Content: prompt})
	return c.GenerateWithMessages(messages, command, system)
}

// GenerateWithMessages sends a conversation history to the AI model and returns the response
func (c *Client) GenerateWithMessages(messages []Message, command string, system string) (Response, error) {
	resp, err := c.provider.GenerateResponse(messages)
	if err != nil {
		return Response{}, err
	}

	if resp.Error != nil {
		return Response{}, resp.Error
	}

	// Calculate cost
	cost, err := c.tracker.CalculateCost(c.model.String(), resp.InputTokens, resp.OutputTokens)
	if err != nil {
		// Log the error but don't fail the request
		fmt.Fprintf(os.Stderr, "Warning: failed to calculate cost: %v\n", err)
		cost = 0
	} else {
		fmt.Fprintf(os.Stderr, "Cost: $%.4f\n", cost)
	}

	// Record stats
	statsTracker, err := stats.NewTracker()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create stats tracker: %v\n", err)
	} else {
		statsTracker.RecordQuery(
			c.model.Provider,
			command,
			resp.InputTokens,
			resp.OutputTokens,
			cost,
			0,
		)
	}

	return resp, nil
}
