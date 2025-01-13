package llmclient

import (
	"net/http"

	"github.com/rs/zerolog"
)

// GeminiProvider implements the Provider interface for Gemini's API.
type GeminiProvider struct {
	BaseProvider
	settings *OpenAISettings
}

// NewGeminiProvider creates a new instance of GeminiProvider.
func NewGeminiProvider(
	model *Model,
	apiKey string,
	client *http.Client,
	apiUrl string,

	logger *zerolog.Logger,
) (*GeminiProvider, error) {
	return &GeminiProvider{
		BaseProvider: *NewBaseProvider(model, apiKey, client, apiUrl, logger),
		settings:     &OpenAISettings{Model: model.Name},
	}, nil
}

func (p *GeminiProvider) Settings() AIModelSettings {
	return p.settings
}

func (p *GeminiProvider) GenerateResponse(messages []AIMessage) (AIResponse, error) {
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
