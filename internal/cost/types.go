package cost

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ModelPricing struct {
	MaxTokens          interface{} `json:"max_tokens"`
	MaxInputTokens     interface{} `json:"max_input_tokens"`
	MaxOutputTokens    interface{} `json:"max_output_tokens"`
	InputCostPerToken  interface{} `json:"input_cost_per_token"`
	OutputCostPerToken interface{} `json:"output_cost_per_token"`
	LiteLLMProvider    string      `json:"litellm_provider"`
	Mode               string      `json:"mode"`
}

func (m *ModelPricing) GetMaxTokens() int {
	switch v := m.MaxTokens.(type) {
	case float64:
		return int(v)
	case string:
		var i int
		fmt.Sscanf(v, "%d", &i)
		return i
	default:
		return 0
	}
}

func (m *ModelPricing) GetMaxInputTokens() int {
	switch v := m.MaxInputTokens.(type) {
	case float64:
		return int(v)
	case string:
		var i int
		fmt.Sscanf(v, "%d", &i)
		return i
	default:
		return 0
	}
}

func (m *ModelPricing) GetMaxOutputTokens() int {
	switch v := m.MaxOutputTokens.(type) {
	case float64:
		return int(v)
	case string:
		var i int
		fmt.Sscanf(v, "%d", &i)
		return i
	default:
		return 0
	}
}

func (m *ModelPricing) GetInputCostPerToken() float64 {
	switch v := m.InputCostPerToken.(type) {
	case float64:
		return v
	case string:
		var f float64
		fmt.Sscanf(v, "%f", &f)
		return f
	default:
		return 0
	}
}

func (m *ModelPricing) GetOutputCostPerToken() float64 {
	switch v := m.OutputCostPerToken.(type) {
	case float64:
		return v
	case string:
		var f float64
		fmt.Sscanf(v, "%f", &f)
		return f
	default:
		return 0
	}
}

// PricingData maps model names to their pricing information
type PricingData map[string]json.RawMessage

// GetModelPricing retrieves and parses pricing data for a specific model
func (p PricingData) GetModelPricing(model string) (*ModelPricing, error) {
	// Try the full model name first
	if raw, ok := p[model]; ok {
		var pricing ModelPricing
		if err := json.Unmarshal(raw, &pricing); err != nil {
			return nil, fmt.Errorf("failed to parse pricing data for model %s: %w", model, err)
		}
		return &pricing, nil
	}

	// Try with just the model name without provider prefix
	parts := strings.Split(model, "/")
	if len(parts) > 1 {
		modelName := parts[len(parts)-1]
		if raw, ok := p[modelName]; ok {
			var pricing ModelPricing
			if err := json.Unmarshal(raw, &pricing); err != nil {
				return nil, fmt.Errorf("failed to parse pricing data for model %s: %w", modelName, err)
			}
			return &pricing, nil
		}
	}

	return nil, fmt.Errorf("no pricing data for model: %s", model)
}
