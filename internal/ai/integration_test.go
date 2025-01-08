package ai

import (
	"os"
	"testing"
)

func TestIntegrationRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	tests := []struct {
		name     string
		model    string
		provider string
		prompt   string
	}{
		{
			name:     "Anthropic Integration",
			model:    "anthropic/claude-2.1",
			provider: "anthropic",
			prompt:   "Say hello in exactly 5 words.",
		},
		{
			name:     "OpenAI Integration",
			model:    "openai/gpt-3.5-turbo",
			provider: "openai",
			prompt:   "Say hello in exactly 5 words.",
		},
		{
			name:     "OpenRouter Integration",
			model:    "openrouter/openai/gpt-3.5-turbo",
			provider: "openrouter",
			prompt:   "Say hello in exactly 5 words.",
		},
		{
			name:     "Gemini Integration",
			model:    "gemini/gemini-pro",
			provider: "gemini",
			prompt:   "Say hello in exactly 5 words.",
		},
		{
			name:     "DeepSeek Integration",
			model:    "deepseek/deepseek-chat",
			provider: "deepseek",
			prompt:   "Say hello in exactly 5 words.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if API key not set
			apiKey := ""
			switch tt.provider {
			case "anthropic":
				apiKey = os.Getenv(EnvAnthropicAPIKey)
			case "openai":
				apiKey = os.Getenv(EnvOpenAIAPIKey)
			case "openrouter":
				apiKey = os.Getenv(EnvOpenRouterAPIKey)
			case "gemini":
				apiKey = os.Getenv(EnvGeminiAPIKey)
			}
			if apiKey == "" {
				t.Skipf("Skipping %s test: no API key set", tt.provider)
			}

			// Set up environment
			os.Setenv(EnvAIModel, tt.model)

			// Create client
			client, err := NewClient()
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// Send request
			response, err := client.Generate(tt.prompt, "test", "")
			if err != nil {
				t.Fatalf("Failed to generate response: %v", err)
			}

			// Basic validation
			if response.Content == "" {
				t.Error("Received empty response")
			}

			t.Logf("Response from %s: %s", tt.provider, response.Content)
		})
	}
}
