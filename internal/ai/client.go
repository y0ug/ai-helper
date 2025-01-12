package ai

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
	GenerateWithMessages(messages []AIMessage, command string) (AIResponse, error)
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

func (c *Client) GetResponseCost(r AIResponse) *float64 {
	var cost *float64
	if c.model.Info != nil {
		inputCost := float64(r.GetUsage().GetInputTokens()) * c.model.Info.InputCostPerToken
		outputCost := float64(r.GetUsage().GetOutputTokens()) * c.model.Info.OutputCostPerToken
		cost = float64ToPtr(inputCost + outputCost)
		// resp.Cost = float64ToPtr(inputCost + outputCost)
	} else {
		fmt.Fprintf(os.Stderr, "Warning: no cost info available for model\n")
	}
	return cost
}

// GenerateWithMessages sends a conversation history to the AI model and returns the response
func (c *Client) GenerateWithMessages(
	messages []AIMessage,
	command string,
) (AIResponse, error) {
	resp, err := c.provider.GenerateResponse(messages)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func float64ToPtr(f float64) *float64 {
	return &f
}

func (client *Client) ProcessMessages(
	messages []AIMessage,
	mcpClient mcpclient.MCPClientInterface,
) ([]AIMessage, error) {
	fmt.Fprintf(os.Stderr, "ProcessMessages\n")
	for _, msg := range messages {
		fmt.Fprintf(os.Stderr, "msg: %T %v\n", msg, msg)
	}

	resp, err := client.GenerateWithMessages(messages, "agent_name")
	if err != nil {
		return nil, err
	}

	choice := resp.GetChoice()
	msg := choice.GetMessage()

	fmt.Fprintf(os.Stderr, "response %v msg: %v\n", resp, msg)
	messages = append(messages, msg)

	fmt.Fprintf(os.Stderr, "choice.GetFinishReason() %s\n", choice.GetFinishReason())

	// if choice.GetFinishReason() == "tool_calls" {
	// Handle tool calls
	var anthropicContent []AIContent

	contents := msg.GetContents()
	for _, content := range contents {
		if content == nil {
			continue
		}

		switch c := content.(type) {
		case AIFunctionCall:
			if c.GetCallType() != "function" {
				fmt.Printf("tool type not supported %s", c.GetCallType())
				continue
			}

			var args map[string]interface{}
			if err := json.Unmarshal([]byte(c.GetArguments()), &args); err != nil {
				return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
			}

			result, err := mcpClient.CallTool(context.Background(), c.GetName(), args)
			if err != nil {
				return nil, fmt.Errorf("failed to call tool %s: %w", c.GetName(), err)
			}

			// Convert result to string
			resultStr := ""
			if result != nil {
				resultBytes, err := json.Marshal(result)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal tool result: %w", err)
				}
				resultStr = string(resultBytes)
			}

			fmt.Fprintf(os.Stderr, "tools: %s: %s\n", c.GetName(), resultStr)

			// Create tool output for OpenAI
			switch content.(type) {
			case OpenAIToolCall:
				msg := OpenAIMessage{
					Role:       "tool",
					Content:    resultStr,
					ToolCallId: c.GetID(),
				}
				messages = append(messages, msg)

			case AnthropicContentToolUse:
				anthropicContent = append(anthropicContent, AnthropicContentToolResult{
					Type:      "tool_result",
					ToolUseId: c.GetID(),
					Content:   resultStr,
				})
				// msg := AnthropicMessageRequest{
				// 	Role: "user",
				// 	Content: AnthropicContentToolResult{
				// 		Type:      "tool_result",
				// 		ToolUseId: c.GetID(),
				// 		Content:   resultStr,
				// 	},
				// }
			}
			if len(anthropicContent) > 0 {
				messages = append(messages, AnthropicMessageRequest{
					Role:    "user",
					Content: anthropicContent,
				})
			}
			messages, err = client.ProcessMessages(messages, mcpClient)
			if err != nil {
				return nil, fmt.Errorf("failed to submit tool outputs: %w", err)
			}
			return messages, nil
		default:
			fmt.Printf("Txt Message: %s\n", c)
		}
	}
	// }
	return messages, nil
}
