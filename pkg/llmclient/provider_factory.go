package llmclient

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog"
)

const (
	anthropicAPIURL  = "https://api.anthropic.com/v1/messages"
	openRouterAPIURL = "https://openrouter.ai/api/v1/chat/completions"
	openAIAPIURL     = "https://api.openai.com/v1/chat/completions"
	geminiAPIURL     = "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"
	deepSeekAPIURL   = "https://api.deepseek.com/v1/chat/completions"
)

// ProviderFactory creates a provider instance based on the model
func NewProvider(
	model *Model,
	apiKey string,
	client *http.Client,
	logger *zerolog.Logger,
) (Provider, error) {
	switch model.Provider {
	case "anthropic":
		return NewAnthropicProvider(model, apiKey, client, anthropicAPIURL, logger)
	case "openrouter":
		return NewOpenRouterProvider(model, apiKey, client, openRouterAPIURL, logger)
	case "openai":
		return NewOpenAIProvider(model, apiKey, client, openAIAPIURL, logger)
	case "gemini":
		return NewGeminiProvider(model, apiKey, client, geminiAPIURL, logger)
	case "deepseek":
		return NewDeepSeekProvider(model, apiKey, client, deepSeekAPIURL, logger)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", model.Provider)
	}
}
