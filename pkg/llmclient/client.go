package llmclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/y0ug/ai-helper/internal/stats"
	"github.com/y0ug/ai-helper/pkg/mcpclient"
)

const (
	EnvAIModel          = "AI_MODEL"
	EnvAnthropicAPIKey  = "ANTHROPIC_API_KEY"
	EnvOpenAIAPIKey     = "OPENAI_API_KEY"
	EnvOpenRouterAPIKey = "OPENROUTER_API_KEY"
	EnvGeminiAPIKey     = "GEMINI_API_KEY"
	EnvDeepSeekAPIKey   = "DEEPSEEK_API_KEY"
)

type AIClient interface {
	SetMaxTokens(maxTokens int)
	SetTools(tools []AITools)
	GenerateWithMessages(messages ...AIMessage) (AIResponse, error)
}

var _ AIClient = (*Client)(nil) // Optional: ensures `Client` implements `AIClient`

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

func (c *Client) SetMaxTokens(maxTokens int) {
	c.provider.Settings().SetMaxTokens(maxTokens)
}

func (c *Client) SetTools(tool []AITools) {
	c.provider.Settings().SetTools(tool)
}

func (c *Client) GetResponseCost(responses ...AIResponse) *float64 {
	if c.model.Info == nil {
		fmt.Fprintf(os.Stderr, "Warning: no cost info available for model\n")
		return nil
	}
	cost := float64(0)
	for _, r := range responses {
		inputCost := float64(r.GetUsage().GetInputTokens()) * c.model.Info.InputCostPerToken
		outputCost := float64(r.GetUsage().GetOutputTokens()) * c.model.Info.OutputCostPerToken
		cost += (inputCost + outputCost)
	}
	return &cost
}

// GenerateWithMessages sends a conversation history to the AI model and returns the response
func (c *Client) GenerateWithMessages(
	messages ...AIMessage,
) (AIResponse, error) {
	resp, err := c.provider.GenerateResponse(messages)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (client *Client) ProcessMessages(
	mcpClient mcpclient.MCPClientInterface,
	messages ...AIMessage,
) ([]AIMessage, []AIResponse, error) {
	responses := make([]AIResponse, 0)
	resp, err := client.GenerateWithMessages(messages...)
	if err != nil {
		return messages, responses, err
	}

	responses = append(responses, resp)

	choice := resp.GetChoice()
	if choice == nil {
		return messages, responses, fmt.Errorf("response choice is nil")
	}

	msg := choice.GetMessage()

	messages = append(messages, msg)
	// fmt.Fprintf(os.Stderr, "choice.GetFinishReason() %s\n", choice.GetFinishReason())

	flush := false

	// if choice.GetFinishReason() == "tool_calls" {
	contents := msg.GetContents()
	for _, content := range contents {
		if content.GetType() == "" {
			continue
		}

		if content.GetType() == string(ContentTypeToolUse) {
			result, err := mcpClient.CallTool(
				context.Background(),
				content.Name,
				content.Input,
			)
			if err != nil {
				return messages, responses, fmt.Errorf(
					"failed to call tool %s: %w",
					content.Name,
					err,
				)
			}

			// Convert result to string
			resultStr := ""
			if result != nil {
				resultBytes, err := json.Marshal(result)
				if err != nil {
					return messages, responses, fmt.Errorf("failed to marshal tool result: %w", err)
				}
				resultStr = string(resultBytes)
			}

			fmt.Fprintf(os.Stderr, "tools: %s: %s\n", content.Name, resultStr)

			// Create tool result message
			toolResultContent := NewToolResultContent(content.ID, resultStr)
			toolResultMsg := NewBaseMessage("user", toolResultContent)
			messages = append(messages, toolResultMsg)
			flush = true

		} else {
			fmt.Printf("Text Message: %s\n", content.String())
		}
	}

	// Recursively process messages, if we have add new
	if flush {
		messages, responses, err = client.ProcessMessages(mcpClient, messages...)
		if err != nil {
			return messages, responses, fmt.Errorf("failed to submit tool outputs: %w", err)
		}
	}
	return messages, responses, nil
}
