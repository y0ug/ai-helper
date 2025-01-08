package ai

import (
	"os"
	"testing"
)

func TestInfoProviders(t *testing.T) {
	// Create a temporary file with valid test data
	tmpFile, err := os.CreateTemp("", "model_info_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write minimal valid JSON data
	testData := `{
		"claude-2.1": {
			"max_tokens": 8191,
			"max_input_tokens": 200000,
			"max_output_tokens": 8191,
			"input_cost_per_token": 0.000008,
			"output_cost_per_token": 0.000024,
			"litellm_provider": "anthropic",
			"mode": "chat"
		},
		"claude-3-haiku-20240307": {
			"max_tokens": 4096,
			"max_input_tokens": 200000,
			"max_output_tokens": 4096,
			"input_cost_per_token": 0.00000025,
			"output_cost_per_token": 0.00000125,
			"cache_creation_input_token_cost": 0.0000003,
			"cache_read_input_token_cost": 0.00000003,
			"litellm_provider": "anthropic",
			"mode": "chat",
			"supports_function_calling": true,
			"supports_vision": true,
			"tool_use_system_prompt_tokens": 264,
			"supports_assistant_prefill": true,
			"supports_prompt_caching": true,
			"supports_response_schema": true
		}
	}`

	if err := os.WriteFile(tmpFile.Name(), []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	// Initialize InfoProviders with temp file
	providers, err := NewInfoProviders(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create InfoProviders: %v", err)
	}

	// Test downloading and loading data
	err = providers.Load()
	if err != nil {
		t.Fatalf("Failed to load model info: %v", err)
	}

	// Test popular models
	testCases := []struct {
		modelName string
		provider  string
	}{
		{"gpt-4", "openai"},
		{"claude-2", "anthropic"},
		{"gemini-pro", "google"},
	}

	for _, tc := range testCases {
		t.Run(tc.modelName, func(t *testing.T) {
			info, err := providers.GetModelInfo(tc.modelName)
			if err != nil {
				t.Errorf("Failed to get info for %s: %v", tc.modelName, err)
				return
			}

			if info == nil {
				t.Errorf("Got nil info for %s", tc.modelName)
				return
			}

			// Check that essential fields have non-zero values
			if info.MaxTokens == 0 {
				t.Errorf("MaxTokens is 0 for %s", tc.modelName)
			}

			if info.InputCostPerToken == 0 {
				t.Errorf("InputCostPerToken is 0 for %s", tc.modelName)
			}

			if info.OutputCostPerToken == 0 {
				t.Errorf("OutputCostPerToken is 0 for %s", tc.modelName)
			}
		})
	}

	// Test provider inference
	t.Run("InferProvider", func(t *testing.T) {
		testCases := []struct {
			modelName        string
			expectedProvider string
		}{
			{"claude-3-opus-20240229", "anthropic"},
			{"gpt-4-turbo-preview", "openai"},
			{"gemini-pro-vision", "google"},
			{"mistral-medium", "mistral"},
			{"llama-2-70b", "meta"},
		}

		for _, tc := range testCases {
			provider := inferProvider(tc.modelName)
			if provider != tc.expectedProvider {
				t.Errorf("Expected provider %s for model %s, got %s",
					tc.expectedProvider, tc.modelName, provider)
			}
		}
	})
}
