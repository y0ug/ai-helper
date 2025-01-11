package ai

import (
	"fmt"
	"net/http"
)

// OpenAIProvider implements the Provider interface for OpenAI's API.
type OpenAIProvider struct {
	BaseProvider
}

// NewOpenAIProvider creates a new instance of OpenAIProvider.
func NewOpenAIProvider(model *Model, apiKey string, client *http.Client) (*OpenAIProvider, error) {
	return &OpenAIProvider{
		BaseProvider: *NewBaseProvider(model, apiKey, client),
	}, nil
}

// OpenAIRequest defines the request structure specific to OpenAI.
type OpenAIRequest struct {
	Model     string    `json:"model"`
	MaxTokens *int      `json:"max_tokens,omitempty"`
	Messages  []Message `json:"messages"`
	Tools     []AITools `json:"tools,omitempty"`
}

// OpenAIResponse defines the response structure specific to OpenAI.
type OpenAIResponse struct {
	ID string `json:"id"`

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

// GenerateResponse sends a request to OpenAI's API and parses the response.
func (p *OpenAIProvider) GenerateResponse(messages []Message) (Response, error) {
	reqPayload := OpenAIRequest{
		Model:    p.model.Name,
		Messages: messages,
	}

	if p.maxTokens != nil {
		reqPayload.MaxTokens = p.maxTokens
	}

	if p.tools != nil {
		reqPayload.Tools = p.tools
	}

	var apiResp OpenAIResponse

	headers := map[string]string{}
	p.setAuthorizationHeader(headers)

	err := p.makeRequest("POST", openAIAPIURL, headers, reqPayload, &apiResp)
	if err != nil {
		return Response{Error: err}, nil
	}

	if len(apiResp.Choices) == 0 {
		return Response{Error: fmt.Errorf("empty response from OpenAI API")}, nil
	}

	return Response{
		Content:      apiResp.Choices[0].Message.Content,
		InputTokens:  apiResp.Usage.PromptTokens,
		OutputTokens: apiResp.Usage.CompletionTokens,
		CachedTokens: apiResp.Usage.PromptTokensDetails.CachedTokens,
	}, nil
}
