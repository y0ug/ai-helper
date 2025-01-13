package llmclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/pkg/mcpclient"
)

func TestFunctionExecution(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		provider string
		prompt   string
	}{
		{
			name:     "Anthropic Integration",
			model:    "anthropic/claude-3-5-sonnet-20241022",
			provider: "anthropic",
			prompt:   "Say hello in exactly 5 words.",
		},
		{
			name:     "OpenAI Integration",
			model:    "openai/gpt-4o",
			provider: "openai",
			prompt:   "Say hello in exactly 5 words.",
		},
		{
			name:     "OpenRouter Integration",
			model:    "openrouter/openai/gpt-3.5-turbo",
			provider: "openrouter",
			prompt:   "Say hello in exactly 5 words.",
		},
		// {
		// 	name:  "Gemini Integration",
		// 	// model: "gemini/gemini-pro",
		// 	model:    "gemini/gemini-exp-1206",
		// 	provider: "gemini",
		// 	prompt:   "Say hello in exactly 5 words.",
		// },
		// {
		// 	name:     "DeepSeek Integration",
		// 	model:    "deepseek/deepseek-chat",
		// 	provider: "deepseek",
		// 	prompt:   "Say hello in exactly 5 words.",
		// },
	}

	config := &config.MCPServer{
		Command: "docker",
		Args:    []string{"run", "--rm", "-i", "mcp/time"},
	}

	// Create new MCP mcpClient
	mcpClient, err := mcpclient.NewMCPClient(config.Command, config.Args...)
	if err != nil {
		t.Fatalf("failed to create MCP client: %s", err)
	}
	defer mcpClient.Close()

	// Initialize the client
	ctx := context.Background()
	if _, err := mcpClient.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize MCP client: %s", err)
	}
	tools, err := mcpclient.FetchAll(context.Background(), mcpClient.ListTools)
	if err != nil {
		t.Fatalf("failed to fetch tools: %s", err)
	}
	aiTools := ToAITools(tools)

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
			case "deepseek":
				apiKey = os.Getenv(EnvDeepSeekAPIKey)
			}
			if apiKey == "" {
				t.Skipf("Skipping %s test: no API key set", tt.provider)
			}

			// Set up environment
			infoProviders, err := NewInfoProviders("")
			if err != nil {
				t.Fatalf("Failed to create info providers: %v", err)
			}
			model, err := ParseModel(tt.model, infoProviders)
			if err != nil {
				t.Fatalf("Failed to get model info: %v", err)
			}

			// Create client
			client, err := NewClient(model, nil)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			client.SetTools(aiTools)

			prompt := "What time is it at Paris?"
			msgReq := NewBaseMessage("user", NewTextContent(prompt))

			// Send request
			messages, _, err := client.ProcessMessages(mcpClient, msgReq)
			if err != nil {
				t.Fatalf("Failed to generate response: %v", err)
			}

			fmt.Println("Messages:")
			for _, mm := range messages {
				data, _ := json.Marshal(mm)
				fmt.Printf("%s\n", data)
			}
		})
	}
}

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
			name:  "Gemini Integration",
			model: "gemini/gemini-pro",
			// model:    "gemini/gemini-exp-1206",
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
			case "deepseek":
				apiKey = os.Getenv(EnvDeepSeekAPIKey)
			}
			if apiKey == "" {
				t.Skipf("Skipping %s test: no API key set", tt.provider)
			}

			// Set up environment
			infoProviders, err := NewInfoProviders("")
			if err != nil {
				t.Fatalf("Failed to create info providers: %v", err)
			}
			model, err := ParseModel(tt.model, infoProviders)
			if err != nil {
				t.Fatalf("Failed to get model info: %v", err)
			}

			// Create client
			client, err := NewClient(model, nil)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			msgReq := NewBaseMessage("user", NewTextContent(tt.prompt))

			// Send request
			response, err := client.GenerateWithMessages(msgReq)
			if err != nil {
				t.Fatalf("Failed to generate response: %v", err)
			}

			// Basic validation
			choice := response.GetChoice()
			msg := choice.GetMessage()
			// if msg.GetContent() == "" {
			// 	t.Error("Received empty response")
			// }

			fmt.Println("Content:")
			for _, v := range msg.GetContents() {
				fmt.Println(v)
			}
			t.Logf("Response from %s: %s", tt.provider, msg.GetContent())
		})
	}
}
