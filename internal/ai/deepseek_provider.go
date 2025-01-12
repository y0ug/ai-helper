package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// DeepSeekProvider implements the Provider interface for DeepSeek's API.
type DeepSeekProvider struct {
	BaseProvider
	settings *OpenAISettings
}

// NewDeepSeekProvider creates a new instance of DeepSeekProvider.
func NewDeepSeekProvider(
	model *Model,
	apiKey string,
	client *http.Client,
	apiUrl string,
) (*DeepSeekProvider, error) {
	return &DeepSeekProvider{
		BaseProvider: *NewBaseProvider(model, apiKey, client, apiUrl),
		settings:     &OpenAISettings{Model: model.Name},
	}, nil
}

func (p *DeepSeekProvider) Settings() AIModelSettings {
	return p.settings
}

// DeepSeekRequest defines the request structure specific to DeepSeek.
type DeepSeekRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

type DeepSeekUsage struct {
	PromptTokens          int `json:"prompt_tokens"`
	CompletionTokens      int `json:"completion_tokens"`
	TotalTokens           int `json:"total_tokens"`
	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens"`
}

type DeepSeekResponse struct {
	OpenAIResponse
	Usage DeepSeekUsage `json:"usage"`
}

func (u DeepSeekUsage) GetInputTokens() int {
	return u.PromptTokens
}

func (u DeepSeekUsage) GetOutputTokens() int {
	return u.CompletionTokens
}

func (u DeepSeekUsage) GetCachedTokens() int {
	return u.PromptCacheHitTokens
}

func (r DeepSeekResponse) GetUsage() AIUsage {
	return r.Usage
}

func (p *DeepSeekProvider) GenerateResponse(messages []AIMessage) (AIResponse, error) {
	headers := map[string]string{}
	p.setAuthorizationHeader(headers)

	req := OpenAIRequest{
		Messages:       messages,
		OpenAISettings: *p.settings,
	}

	for _, msg := range req.Messages {
		data, _ := json.Marshal(msg)
		fmt.Fprintf(os.Stderr, "msg: %T %v\n", msg, string(data))
	}
	var resp DeepSeekResponse
	err := p.makeRequest("POST", p.baseUrl, headers, req, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
