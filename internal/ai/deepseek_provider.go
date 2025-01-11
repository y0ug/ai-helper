package ai

import (
	"fmt"
	"net/http"
)

// DeepSeekProvider implements the Provider interface for DeepSeek's API.
type DeepSeekProvider struct {
	BaseProvider
}

// NewDeepSeekProvider creates a new instance of DeepSeekProvider.
func NewDeepSeekProvider(
	model *Model,
	apiKey string,
	client *http.Client,
) (*DeepSeekProvider, error) {
	return &DeepSeekProvider{
		BaseProvider: *NewBaseProvider(model, apiKey, client),
	}, nil
}

// DeepSeekRequest defines the request structure specific to DeepSeek.
type DeepSeekRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

// DeepSeekResponse defines the response structure specific to DeepSeek.
type DeepSeekResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens          int `json:"prompt_tokens"`
		CompletionTokens      int `json:"completion_tokens"`
		TotalTokens           int `json:"total_tokens"`
		PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens"`
		PromptCacheMissTokens int `json:"prompt_cache_miss_tokens"`
	} `json:"usage"`
}

// GenerateResponse sends a request to DeepSeek's API and parses the response.
func (p *DeepSeekProvider) GenerateResponse(messages []Message) (Response, error) {
	reqPayload := DeepSeekRequest{
		Model:     p.model.Name,
		MaxTokens: 1024,
		Messages:  messages,
	}

	var apiResp DeepSeekResponse

	headers := map[string]string{}
	p.setAuthorizationHeader(headers)

	err := p.makeRequest("POST", deepSeekAPIURL, headers, reqPayload, &apiResp)
	if err != nil {
		return Response{Error: err}, nil
	}

	if len(apiResp.Choices) == 0 {
		return Response{Error: fmt.Errorf("empty response from DeepSeek API")}, nil
	}

	return Response{
		Content:      apiResp.Choices[0].Message.Content,
		InputTokens:  apiResp.Usage.PromptTokens,
		OutputTokens: apiResp.Usage.CompletionTokens,
		CachedTokens: apiResp.Usage.PromptCacheHitTokens,
	}, nil
}
