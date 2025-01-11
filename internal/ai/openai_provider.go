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
	ID     string `json:"id"`
	Status string `json:"status,omitempty"`

	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`

	RequiredAction *struct {
		SubmitToolOutputs struct {
			ToolCalls []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
				Type string `json:"type"`
			} `json:"tool_calls"`
		} `json:"submit_tool_outputs"`
		Type string `json:"type"`
	} `json:"required_action,omitempty"`

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

	if apiResp.Status == "requires_action" && apiResp.RequiredAction != nil {
		// Handle function calls
		var toolCalls []ToolCall
		for _, call := range apiResp.RequiredAction.SubmitToolOutputs.ToolCalls {
			toolCalls = append(toolCalls, ToolCall{
				ID:       call.ID,
				Name:     call.Function.Name,
				Args:     call.Function.Arguments,
				Type:     call.Type,
			})
		}
		return Response{
			RequiresAction: true,
			ToolCalls:      toolCalls,
			InputTokens:    apiResp.Usage.PromptTokens,
			OutputTokens:   apiResp.Usage.CompletionTokens,
			CachedTokens:   apiResp.Usage.PromptTokensDetails.CachedTokens,
		}, nil
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
