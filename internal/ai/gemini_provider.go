package ai

import (
	"fmt"
	"net/http"
)

// GeminiProvider implements the Provider interface for Gemini's API.
type GeminiProvider struct {
	BaseProvider
}

// NewGeminiProvider creates a new instance of GeminiProvider.
func NewGeminiProvider(model *Model, apiKey string, client *http.Client) (*GeminiProvider, error) {
	return &GeminiProvider{
		BaseProvider: *NewBaseProvider(model, apiKey, client),
	}, nil
}

// GeminiRequest defines the request structure using OpenAI compatibility mode
type GeminiRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

// GeminiResponse defines the response structure using OpenAI compatibility mode
type GeminiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// GenerateResponse sends a request to Gemini's API using OpenAI compatibility mode
func (p *GeminiProvider) GenerateResponse(messages []Message) (Response, error) {
	reqPayload := GeminiRequest{
		Model:     p.model.Name,
		MaxTokens: 1024,
		Messages:  messages,
	}

	var apiResp GeminiResponse

	headers := map[string]string{}
	p.setAuthorizationHeader(headers)

	err := p.makeRequest("POST", geminiAPIURL, headers, reqPayload, &apiResp)
	if err != nil {
		return Response{Error: err}, nil
	}

	if len(apiResp.Choices) == 0 {
		return Response{Error: fmt.Errorf("empty response from Gemini API")}, nil
	}

	return Response{
		Content:      apiResp.Choices[0].Message.Content,
		InputTokens:  apiResp.Usage.PromptTokens,
		OutputTokens: apiResp.Usage.CompletionTokens,
	}, nil
}
