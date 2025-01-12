package ai

import (
	"fmt"
	"net/http"
)

const (
	anthropicAPIURL  = "https://api.anthropic.com/v1/messages"
	openRouterAPIURL = "https://openrouter.ai/api/v1/chat/completions"
	openAIAPIURL     = "https://api.openai.com/v1/chat/completions"
	geminiAPIURL     = "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"
	deepSeekAPIURL   = "https://api.deepseek.com/v1/chat/completions"
)

// ProviderFactory creates a provider instance based on the model
func NewProvider(model *Model, apiKey string, client *http.Client) (Provider, error) {
	switch model.Provider {
	case "anthropic":
		return NewAnthropicProvider(model, apiKey, client, anthropicAPIURL)
	case "openrouter":
		return NewOpenRouterProvider(model, apiKey, client, openRouterAPIURL)
	case "openai":
		return NewOpenAIProvider(model, apiKey, client, openAIAPIURL)
	case "gemini":
		return NewGeminiProvider(model, apiKey, client, geminiAPIURL)
	case "deepseek":
		return NewDeepSeekProvider(model, apiKey, client, deepSeekAPIURL)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", model.Provider)
	}
}
