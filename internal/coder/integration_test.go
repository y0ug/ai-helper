package coder

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/y0ug/ai-helper/internal/ai"
	"github.com/y0ug/ai-helper/internal/coder/prompts"
)

func TestCoder_Integration(t *testing.T) {
	testCases := []struct {
		name      string
		modelStr  string
		envKey    string
		envKeyMsg string
	}{
		{
			name:      "Claude Integration",
			modelStr:  "claude-3-5-sonnet-20241022",
			envKey:    "ANTHROPIC_API_KEY",
			envKeyMsg: "ANTHROPIC_API_KEY environment variable must be set",
		},
		{
			name:      "DeepSeek Integration",
			modelStr:  "deepseek-chat",
			envKey:    "DEEPSEEK_API_KEY",
			envKeyMsg: "DEEPSEEK_API_KEY environment variable must be set",
		},
		{
			name:      "GPT-4 Integration",
			modelStr:  "openai/gpt-4",
			envKey:    "OPENAI_API_KEY",
			envKeyMsg: "OPENAI_API_KEY environment variable must be set",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Skip if not running integration tests
			if os.Getenv("INTEGRATION_TESTS") != "1" {
				t.Skip("Skipping integration test. Set INTEGRATION_TESTS=1 to run")
			}

			prompts.ResetTemplatesFS()

			// Ensure API key is set
			apiKey := os.Getenv(tc.envKey)
			require.NotEmpty(t, apiKey, tc.envKeyMsg)

			// Create client
			infoProviders, err := ai.NewInfoProviders("")
			if err != nil {
				t.Fatalf("Failed to create info providers: %v", err)
			}
			model, err := ai.ParseModel(tc.modelStr, infoProviders)
			if err != nil {
				t.Fatalf("Failed to get model info: %v", err)
			}

			// Create client
			client, err := ai.NewClient(model, nil)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// Create test agent with real client
			agent := &ai.Agent{
				Client:   client,
				Messages: []ai.Message{},
			}

			// Test files
			files := map[string]string{
				"example.go": `package main

func main() {
	// TODO: Implement proper greeting with Hello, World!
	println("hi")
}`,
			}

			// Create coder instance
			coder := New(agent)
			coder.SetTemplateData(files)
			//
			// Test request
			resp, err := coder.RequestChange(context.Background(),
				// "Update the greeting to print 'Hello, World!' with proper formatting",
				"Apply the TODO and remove it.",
				files)

			// Assertions
			require.NoError(t, err)
			require.NotNil(t, resp)

			// Verify the changes
			assert.Contains(t, resp.ModifiedFiles["example.go"], "Hello, World!")
		})
	}
}
