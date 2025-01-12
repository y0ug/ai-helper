package ai

import (
	"fmt"
	"net/http"
)

// AnthropicProvider implements the Provider interface for Anthropic's API.
type AnthropicProvider struct {
	BaseProvider
}

// NewAnthropicProvider creates a new instance of AnthropicProvider.
func NewAnthropicProvider(
	model *Model,
	apiKey string,
	client *http.Client,
	apiUrl string,
) (*AnthropicProvider, error) {
	return &AnthropicProvider{
		BaseProvider: *NewBaseProvider(model, apiKey, client, nil, apiUrl),
	}, nil
}

// AnthropicRequest defines the request structure specific to Anthropic.
type AnthropicRequest struct {
	Model     string    `json:"model"`
	System    string    `json:"system,omitempty"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

// AnthropicResponse defines the response structure specific to Anthropic.
type AnthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// GenerateResponse sends a request to Anthropic's API and parses the response.
func (p *AnthropicProvider) GenerateResponse(messages []Message) (Response, error) {
	var systemPrompt string
	var userMessages []Message

	for _, msg := range messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
		} else {
			userMessages = append(userMessages, msg)
		}
	}

	reqPayload := AnthropicRequest{
		Model:     p.model.Name,
		System:    systemPrompt,
		MaxTokens: 1024,
		Messages:  userMessages,
	}

	var apiResp AnthropicResponse

	headers := map[string]string{
		"anthropic-version": "2023-06-01",
		"x-api-key":         p.apiKey,
	}

	err := p.makeRequest("POST", anthropicAPIURL, headers, reqPayload, &apiResp)
	if err != nil {
		return Response{Error: err}, nil
	}

	if len(apiResp.Content) == 0 {
		return Response{Error: fmt.Errorf("empty response from Anthropic API")}, nil
	}

	return Response{
		Content:      apiResp.Content[0].Text,
		InputTokens:  apiResp.Usage.InputTokens,
		OutputTokens: apiResp.Usage.OutputTokens,
	}, nil
}
