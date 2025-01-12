package ai

import (
	"net/http"
)

// OpenRouterProvider implements the Provider interface for OpenRouter's API.
type OpenRouterProvider struct {
	BaseProvider
	settings *OpenAISettings
}

// NewOpenRouterProvider creates a new instance of OpenRouterProvider.
func NewOpenRouterProvider(
	model *Model,
	apiKey string,
	client *http.Client,
	apiUrl string,
) (*OpenRouterProvider, error) {
	return &OpenRouterProvider{
		BaseProvider: *NewBaseProvider(model, apiKey, client, apiUrl),
		settings:     &OpenAISettings{Model: model.Name},
	}, nil
}

func (p *OpenRouterProvider) Settings() AIModelSettings {
	return p.settings
}

func (p *OpenRouterProvider) GenerateResponse(messages []AIMessage) (AIResponse, error) {
	headers := map[string]string{}
	p.setAuthorizationHeader(headers)

	req := OpenAIRequest{
		Messages:       AIMessageToOpenAIMessage(messages),
		OpenAISettings: *p.settings,
	}

	var resp OpenAIResponse
	err := p.makeRequest("POST", p.baseUrl, headers, req, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
