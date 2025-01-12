package ai

import (
	"fmt"
	"net/http"
)

// OpenRouterProvider implements the Provider interface for OpenRouter's API.
type OpenRouterProvider struct {
	BaseProvider
	settings OpenAISettings
}

// NewOpenRouterProvider creates a new instance of OpenRouterProvider.
func NewOpenRouterProvider(
	model *Model,
	apiKey string,
	client *http.Client,
	apiUrl string,
) (*OpenRouterProvider, error) {
	settings := OpenAISettings{
		Model: model.Name,
	}
	return &OpenRouterProvider{
		BaseProvider: *NewBaseProvider(model, apiKey, client, settings, apiUrl),
	}, nil
}

func (p *OpenRouterProvider) GenerateResponse(messages []AIMessage) (AIResponse, error) {
	headers := map[string]string{}
	p.setAuthorizationHeader(headers)

	req := OpenAIRequest{
		Messages:       messages,
		OpenAISettings: p.settings,
	}

	fmt.Println(p.model.Name)
	var resp OpenAIResponse
	err := p.makeRequest("POST", p.baseUrl, headers, req, &resp)
	if err != nil {
		return nil, err
	}

	fmt.Println(resp)
	return resp, nil
}
