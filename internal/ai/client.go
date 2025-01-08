package ai

import (
	"fmt"
	"os"

	"github.com/y0ug/ai-helper/internal/stats"
)

const (
	EnvAIModel          = "AI_MODEL"
	EnvAnthropicAPIKey  = "ANTHROPIC_API_KEY"
	EnvOpenAIAPIKey     = "OPENAI_API_KEY"
	EnvOpenRouterAPIKey = "OPENROUTER_API_KEY"
	EnvGeminiAPIKey     = "GEMINI_API_KEY"
	EnvDeepSeekAPIKey   = "DEEPSEEK_API_KEY"
)

// Client handles AI model interactions
type Client struct {
	provider Provider
	model    *Model
	stats    *stats.Tracker
}

// NewClient creates a new AI client using environment variables
func NewClient(
	model *Model,
	statsTracker *stats.Tracker,
) (*Client, error) {
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
	case "deepseek":
		apiKey = os.Getenv(EnvDeepSeekAPIKey)
		if apiKey == "" {
			return nil, fmt.Errorf("DEEPSEEK_API_KEY environment variable not set")
		}
	default:
		return nil, fmt.Errorf("unsupported provider: %s", model.Provider)
	}

	provider, err := NewProvider(model, apiKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return &Client{
		provider: provider,
		model:    model,
		stats:    statsTracker,
	}, nil
}

// GenerateWithMessages sends a conversation history to the AI model and returns the response
func (c *Client) GenerateWithMessages(
	messages []Message,
	command string,
) (Response, error) {
	resp, err := c.provider.GenerateResponse(messages)
	if err != nil {
		return Response{}, err
	}

	if resp.Error != nil {
		return Response{}, resp.Error
	}

	// Calculate cost using model info
	if c.model.info != nil {
		inputCost := float64(resp.InputTokens) * c.model.info.InputCostPerToken
		outputCost := float64(resp.OutputTokens) * c.model.info.OutputCostPerToken
		resp.Cost = float64ToPtr(inputCost + outputCost)
	} else {
		fmt.Fprintf(os.Stderr, "Warning: no cost info available for model\n")
	}

	// Record stats
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create stats tracker: %v\n", err)
	} else {
		// TODO: find a better way to handle no cost info available
		cost := 0.0
		if resp.Cost != nil {
			cost = *resp.Cost
		}
		if c.stats != nil {
			c.stats.RecordQuery(
				c.model.Provider,
				command,
				resp.InputTokens,
				resp.OutputTokens,
				cost,
				0,
			)
		}
	}

	return resp, nil
}

func float64ToPtr(f float64) *float64 {
	return &f
}
