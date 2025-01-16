package llmclient

import (
	"fmt"
	"os"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestInfoProviders(t *testing.T) {
	t.Run("Local File", func(t *testing.T) {
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

		// Setup mock controller
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Initialize InfoProviders with temp file
		providers, err := NewModelInfoProvider(tmpFile.Name())
		if err != nil {
			t.Fatalf("Failed to create InfoProviders: %v", err)
		}

		// Test downloading and loading data
		err = providers.Load()
		if err != nil {
			t.Fatalf("Failed to load model info: %v", err)
		}

		// Test models from our test data
		testCases := []struct {
			modelName string
			provider  string
		}{
			{"claude-2.1", "anthropic"},
			{"claude-3-haiku-20240307", "anthropic"},
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
	})

	t.Run("Download Integration", func(t *testing.T) {
		// Create a temporary file for downloaded data
		tmpFile, err := os.CreateTemp("", "model_info_download_*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		// Initialize InfoProviders with temp file
		providers, err := NewModelInfoProvider(tmpFile.Name())
		if err != nil {
			t.Fatalf("Failed to create InfoProviders: %v", err)
		}

		// Force a fresh download
		if err := providers.Clear(); err != nil {
			t.Fatalf("Failed to clear providers: %v", err)
		}

		// Load should trigger a download
		if err := providers.Load(); err != nil {
			t.Fatalf("Failed to load/download model info: %v", err)
		}

		// Test some well-known models that should be in the downloaded data
		knownModels := []struct {
			name     string
			provider string
		}{
			{"gpt-4", "openai"},
			{"claude-2", "anthropic"},
			{"gemini-pro", "vertex_ai-language-models"},
		}

		for _, model := range knownModels {
			info, err := providers.GetModelInfo(fmt.Sprintf("%s/%s", model.provider, model.name))
			if err != nil {
				t.Errorf("Failed to get info for %s: %v", model.name, err)
				continue
			}

			if info.LiteLLMProvider != model.provider {
				t.Errorf("Expected provider %s for model %s, got %s",
					model.provider, model.name, info.LiteLLMProvider)
			}

			// Verify essential fields
			if info.MaxTokens == 0 {
				t.Errorf("MaxTokens is 0 for %s", model.name)
			}
			if info.InputCostPerToken == 0 {
				t.Errorf("InputCostPerToken is 0 for %s", model.name)
			}
			if info.OutputCostPerToken == 0 {
				t.Errorf("OutputCostPerToken is 0 for %s", model.name)
			}
		}
	})

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
