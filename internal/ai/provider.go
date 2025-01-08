package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	anthropicAPIURL  = "https://api.anthropic.com/v1/messages"
	openRouterAPIURL = "https://openrouter.ai/api/v1/chat/completions"
	openAIAPIURL     = "https://api.openai.com/v1/chat/completions"
	geminiAPIURL     = "https://generativelanguage.googleapis.com/v1beta/models"
)

// Provider represents an AI service provider
type Provider interface {
	GenerateResponse(messages []Message) (Response, error)
}

// GeminiProvider implements the Provider interface for Google's Gemini API
type GeminiProvider struct {
	model  string
	apiKey string
	client *http.Client
}

func NewGeminiProvider(model, apiKey string) (*GeminiProvider, error) {
	return &GeminiProvider{
		model:  model,
		apiKey: apiKey,
		client: &http.Client{},
	}, nil
}

func (p *GeminiProvider) GenerateResponse(messages []Message) (Response, error) {
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiAPIURL, p.model, p.apiKey)
	
	req := struct {
		Contents []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"contents"`
	}{
		Contents: []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		}{
			{
				Parts: []struct {
					Text string `json:"text"`
				}{
					{Text: messages[len(messages)-1].Content},
				},
			},
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return Response{Error: fmt.Errorf("failed to marshal request: %w", err)}, nil
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return Response{Error: fmt.Errorf("failed to create request: %w", err)}, nil
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{Error: fmt.Errorf("failed to send request: %w", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Response{
			Error: fmt.Errorf(
				"API request failed with status %d: %s",
				resp.StatusCode,
				string(body),
			),
		}, nil
	}

	var response struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return Response{Error: fmt.Errorf("failed to decode response: %w", err)}, nil
	}

	if len(response.Candidates) == 0 {
		return Response{Error: fmt.Errorf("empty response from API")}, nil
	}

	return Response{
		Content:      response.Candidates[0].Content.Parts[0].Text,
		InputTokens:  response.UsageMetadata.PromptTokenCount,
		OutputTokens: response.UsageMetadata.CandidatesTokenCount,
	}, nil
}

// OpenAIProvider implements the Provider interface for OpenAI's API
type OpenAIProvider struct {
	model  string
	apiKey string
	client *http.Client
}

func NewOpenAIProvider(model, apiKey string) (*OpenAIProvider, error) {
	return &OpenAIProvider{
		model:  model,
		apiKey: apiKey,
		client: &http.Client{},
	}, nil
}

func (p *OpenAIProvider) GenerateResponse(messages []Message) (Response, error) {
	req := struct {
		Model     string    `json:"model"`
		MaxTokens int       `json:"max_tokens"`
		Messages  []Message `json:"messages"`
	}{
		Model:     p.model,
		MaxTokens: 1024,
		Messages: messages,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return Response{Error: fmt.Errorf("failed to marshal request: %w", err)}, nil
	}

	httpReq, err := http.NewRequest("POST", openAIAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return Response{Error: fmt.Errorf("failed to create request: %w", err)}, nil
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{Error: fmt.Errorf("failed to send request: %w", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Response{
			Error: fmt.Errorf(
				"API request failed with status %d: %s",
				resp.StatusCode,
				string(body),
			),
		}, nil
	}

	var response APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return Response{Error: fmt.Errorf("failed to decode response: %w", err)}, nil
	}


	return Response{
		Content:      response.Choices[0].Message.Content,
		InputTokens:  response.Usage.PromptTokens,
		OutputTokens: response.Usage.CompletionTokens,
	}, nil
}

// ProviderFactory creates a provider instance based on the model
func NewProvider(model *Model, apiKey string) (Provider, error) {
	switch model.Provider {
	case "anthropic":
		return NewAnthropicProvider(model.Name, apiKey)
	case "openrouter":
		return NewOpenRouterProvider(model.Name, apiKey)
	case "openai":
		return NewOpenAIProvider(model.Name, apiKey)
	case "gemini":
		return NewGeminiProvider(model.Name, apiKey)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", model.Provider)
	}
}

// AnthropicProvider implements the Provider interface for Anthropic's API
type AnthropicProvider struct {
	model  string
	apiKey string
	client *http.Client
}

func NewAnthropicProvider(model, apiKey string) (*AnthropicProvider, error) {
	return &AnthropicProvider{
		model:  model,
		apiKey: apiKey,
		client: &http.Client{},
	}, nil
}

func (p *AnthropicProvider) GenerateResponse(messages []Message) (Response, error) {
	req := struct {
		Model     string    `json:"model"`
		MaxTokens int       `json:"max_tokens"`
		Messages  []Message `json:"messages"`
	}{
		Model:     p.model,
		MaxTokens: 1024,
		Messages: messages,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return Response{Error: fmt.Errorf("failed to marshal request: %w", err)}, nil
	}

	httpReq, err := http.NewRequest("POST", anthropicAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return Response{Error: fmt.Errorf("failed to create request: %w", err)}, nil
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{Error: fmt.Errorf("failed to send request: %w", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Response{
			Error: fmt.Errorf(
				"API request failed with status %d: %s",
				resp.StatusCode,
				string(body),
			),
		}, nil
	}

	var response AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return Response{Error: fmt.Errorf("failed to decode response: %w", err)}, nil
	}


	if len(response.Content) == 0 {
		return Response{Error: fmt.Errorf("empty response from API")}, nil
	}

	return Response{
		Content:      response.Content[0].Text,
		InputTokens:  response.Usage.InputTokens,
		OutputTokens: response.Usage.OutputTokens,
	}, nil
}

// OpenRouterProvider implements the Provider interface for OpenRouter's API
type OpenRouterProvider struct {
	model  string
	apiKey string
	client *http.Client
}

func NewOpenRouterProvider(model, apiKey string) (*OpenRouterProvider, error) {
	return &OpenRouterProvider{
		model:  model,
		apiKey: apiKey,
		client: &http.Client{},
	}, nil
}

func (p *OpenRouterProvider) GenerateResponse(messages []Message) (Response, error) {
	req := Request{
		Model:    p.model,
		Messages: messages,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return Response{Error: fmt.Errorf("failed to marshal request: %w", err)}, nil
	}

	httpReq, err := http.NewRequest("POST", openRouterAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return Response{Error: fmt.Errorf("failed to create request: %w", err)}, nil
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://github.com/yourusername/ai-helper")
	httpReq.Header.Set("X-Title", "AI Helper CLI")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{Error: fmt.Errorf("failed to send request: %w", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Response{
			Error: fmt.Errorf(
				"API request failed with status %d: %s",
				resp.StatusCode,
				string(body),
			),
		}, nil
	}

	var response APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return Response{Error: fmt.Errorf("failed to decode response: %w", err)}, nil
	}

	if len(response.Choices) == 0 {
		return Response{Error: fmt.Errorf("empty response from API")}, nil
	}

	return Response{
		Content:      response.Choices[0].Message.Content,
		InputTokens:  response.Usage.PromptTokens,
		OutputTokens: response.Usage.CompletionTokens,
	}, nil
}
