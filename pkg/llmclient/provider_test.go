package llmclient

import (
	"os"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestProviderEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		apiKey   string
		wantErr  bool
		provider string
	}{
		{
			name:     "Anthropic Provider",
			model:    "anthropic/claude-2.1",
			apiKey:   "test-anthropic-key",
			wantErr:  false,
			provider: "anthropic",
		},
		{
			name:     "OpenRouter Provider",
			model:    "openrouter/openai/gpt-4",
			apiKey:   "test-openrouter-key",
			wantErr:  false,
			provider: "openrouter",
		},
		{
			name:     "OpenAI Provider",
			model:    "openai/gpt-4",
			apiKey:   "test-openai-key",
			wantErr:  false,
			provider: "openai",
		},
		{
			name:     "Gemini Provider",
			model:    "gemini/gemini-pro",
			apiKey:   "test-gemini-key",
			wantErr:  false,
			provider: "gemini",
		},
		{
			name:     "DeepSeek Provider",
			model:    "deepseek/deepseek-chat",
			apiKey:   "test-deepseek-key",
			wantErr:  false,
			provider: "deepseek",
		},
		{
			name:     "Invalid Provider",
			model:    "invalid/model",
			apiKey:   "test-key",
			wantErr:  true,
			provider: "invalid",
		},
		{
			name:     "Empty API Key",
			model:    "openai/gpt-4",
			apiKey:   "",
			wantErr:  true,
			provider: "openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv(EnvAIModel, tt.model)
			switch tt.provider {
			case "anthropic":
				os.Setenv(EnvAnthropicAPIKey, tt.apiKey)
			case "openai":
				os.Setenv(EnvOpenAIAPIKey, tt.apiKey)
			case "openrouter":
				os.Setenv(EnvOpenRouterAPIKey, tt.apiKey)
			case "gemini":
				os.Setenv(EnvGeminiAPIKey, tt.apiKey)
			case "deepseek":
				os.Setenv(EnvDeepSeekAPIKey, tt.apiKey)
			}

			// Parse and create model
			model, err := ParseModel(tt.model, nil)
			if err != nil {
				t.Fatalf("Failed to parse model: %v", err)
			}

			// Setup mock controller
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock client
			_ = NewMockAIClient(ctrl)

			// Create new client
			client, err := NewClient(model, nil, &testLogger)

			// Check error cases
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewClient() error = nil, wantErr = true")
				}
				return
			}

			if err != nil {
				t.Errorf("NewClient() unexpected error = %v", err)
				return
			}

			// Verify provider type
			if client.model.Provider != tt.provider {
				t.Errorf("Provider = %v, want %v", client.model.Provider, tt.provider)
			}
		})
	}

	// Clean up environment variables after tests
	os.Unsetenv(EnvAIModel)
	os.Unsetenv(EnvAnthropicAPIKey)
	os.Unsetenv(EnvOpenAIAPIKey)
	os.Unsetenv(EnvOpenRouterAPIKey)
	os.Unsetenv(EnvDeepSeekAPIKey)
}

func TestModelParsing(t *testing.T) {
	tests := []struct {
		name     string
		modelStr string
		wantErr  bool
		provider string
		model    string
	}{
		{
			name:     "Valid Anthropic Model",
			modelStr: "anthropic/claude-2.1",
			wantErr:  false,
			provider: "anthropic",
			model:    "claude-2.1",
		},
		{
			name:     "Valid OpenRouter Model",
			modelStr: "openrouter/openai/gpt-4",
			wantErr:  false,
			provider: "openrouter",
			model:    "openai/gpt-4",
		},
		{
			name:     "Invalid Format",
			modelStr: "invalid-model",
			wantErr:  true,
		},
		{
			name:     "Empty String",
			modelStr: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := ParseModel(tt.modelStr, nil)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseModel() error = nil, wantErr = true")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseModel() unexpected error = %v", err)
				return
			}

			if model.Provider != tt.provider {
				t.Errorf("Provider = %v, want %v", model.Provider, tt.provider)
			}

			if model.Name != tt.model {
				t.Errorf("Model = %v, want %v", model.Name, tt.model)
			}
		})
	}
}
