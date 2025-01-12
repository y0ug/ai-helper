package ai

import (
	"fmt"
	"net/http"
)

// GeminiProvider implements the Provider interface for Gemini's API.
type GeminiProvider struct {
	BaseProvider
	settings OpenAISettings
}

// NewGeminiProvider creates a new instance of GeminiProvider.
func NewGeminiProvider(
	model *Model,
	apiKey string,
	client *http.Client,
	apiUrl string,
) (*GeminiProvider, error) {
	settings := OpenAISettings{
		Model: model.Name,
	}
	return &GeminiProvider{
		BaseProvider: *NewBaseProvider(model, apiKey, client, settings, apiUrl),
	}, nil
}

func (p *GeminiProvider) GenerateResponse(messages []AIMessage) (AIResponse, error) {
	headers := map[string]string{}
	p.setAuthorizationHeader(headers)

	req := OpenAIRequest{
		Messages:       messages,
		OpenAISettings: p.settings,
	}

	var resp OpenAIResponse
	err := p.makeRequest("POST", p.baseUrl, headers, req, &resp)
	if err != nil {
		return nil, err
	}

	fmt.Println(resp)
	return resp, nil
}
