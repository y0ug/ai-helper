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

// GeminiRequest defines the request structure specific to Gemini.
type GeminiParts struct {
  Test string `json:"text"`
}

type GeminiContents struct {
  Contents []GeminiParts `json:"parts"`
}

type GeminiRequest struct {
  GeminiContents `json:"contents"`
}

// GeminiResponse defines the response structure specific to Gemini.
type GeminiResponse struct {
	Choices []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"choices"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

// GenerateResponse sends a request to Gemini's API and parses the response.
func (p *GeminiProvider) GenerateResponse(messages []Message) (Response, error) {
	reqPayload := GeminiRequest{
    contents = [
      Parts = {

  ]
	}

	var apiResp GeminiResponse

	headers := map[string]string{}
	// p.setAuthorizationHeader(headers)
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiAPIURL, p.model.Name, p.apiKey)
	err := p.makeRequest("POST", url, headers, reqPayload, &apiResp)
	if err != nil {
		return Response{Error: err}, nil
	}

	if len(apiResp.Choices) == 0 || len(apiResp.Choices[0].Content.Parts) == 0 {
		return Response{Error: fmt.Errorf("empty response from Gemini API")}, nil
	}

	return Response{
		Content:      apiResp.Choices[0].Content.Parts[0].Text,
		InputTokens:  apiResp.UsageMetadata.PromptTokenCount,
		OutputTokens: apiResp.UsageMetadata.CandidatesTokenCount,
	}, nil
}
