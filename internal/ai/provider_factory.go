package ai

import (
	"fmt"
	"net/http"
)

const (
	anthropicAPIURL  = "https://api.anthropic.com/v1/messages"
	openRouterAPIURL = "https://openrouter.ai/api/v1/chat/completions"
	openAIAPIURL     = "https://api.openai.com/v1/chat/completions"
	geminiAPIURL     = "https://generativelanguage.googleapis.com/v1beta/models"
	deepSeekAPIURL   = "https://api.deepseek.com/v1/chat/completions"
)

// ProviderFactory creates a provider instance based on the model
func NewProvider(model *Model, apiKey string, client *http.Client) (Provider, error) {
	switch model.Provider {
	case "anthropic":
		return NewAnthropicProvider(model, apiKey, client)
	case "openrouter":
		return NewOpenRouterProvider(model, apiKey, client)
	case "openai":
		return NewOpenAIProvider(model, apiKey, client)
	case "gemini":
		return NewGeminiProvider(model, apiKey, client)
	case "deepseek":
		return NewDeepSeekProvider(model, apiKey, client)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", model.Provider)
	}
}
