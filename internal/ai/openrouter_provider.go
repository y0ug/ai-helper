package ai

import (
	"fmt"
	"net/http"
)

// OpenRouterProvider implements the Provider interface for OpenRouter's API.
type OpenRouterProvider struct {
	BaseProvider
}

// NewOpenRouterProvider creates a new instance of OpenRouterProvider.
func NewOpenRouterProvider(
	model *Model,
	apiKey string,
	client *http.Client,
) (*OpenRouterProvider, error) {
	return &OpenRouterProvider{
		BaseProvider: *NewBaseProvider(model, apiKey, client),
	}, nil
}

// OpenRouterRequest defines the request structure specific to OpenRouter.
type OpenRouterRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

// OpenRouterResponse defines the response structure specific to OpenRouter.
type OpenRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		TotalTokens         int `json:"total_tokens"`
		PromptTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
	} `json:"usage"`
}

// GenerateResponse sends a request to OpenRouter's API and parses the response.
func (p *OpenRouterProvider) GenerateResponse(messages []Message) (Response, error) {
	reqPayload := OpenRouterRequest{
		Model:     p.model.Name,
		MaxTokens: 1024,
		Messages:  messages,
	}

	var apiResp OpenRouterResponse

	headers := map[string]string{}
	p.setAuthorizationHeader(headers)

	err := p.makeRequest("POST", openRouterAPIURL, headers, reqPayload, &apiResp)
	if err != nil {
		return Response{Error: err}, nil
	}

	if len(apiResp.Choices) == 0 {
		return Response{Error: fmt.Errorf("empty response from OpenRouter API")}, nil
	}

	return Response{
		Content:      apiResp.Choices[0].Message.Content,
		InputTokens:  apiResp.Usage.PromptTokens,
		OutputTokens: apiResp.Usage.CompletionTokens,
		CachedTokens: apiResp.Usage.PromptTokensDetails.CachedTokens,
	}, nil
}
