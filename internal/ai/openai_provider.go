package ai

import (
	"fmt"
	"net/http"
)

// OpenAIProvider implements the Provider interface for OpenAI's API.
type OpenAIProvider struct {
	BaseProvider
	settings OpenAISettings
}

type OpenAIResponseFormat struct {
	Type       string      `json:"type"`
	JsonSchema interface{} `json:"json_schema"`
}

// NewOpenAIProvider creates a new instance of OpenAIProvider.
func NewOpenAIProvider(
	model *Model,
	apiKey string,
	client *http.Client,
	apiUrl string,
) (*OpenAIProvider, error) {
	settings := OpenAISettings{
		Model: model.Name,
	}

	return &OpenAIProvider{
		BaseProvider: *NewBaseProvider(model, apiKey, client, &settings, apiUrl),
	}, nil
}

type OpenAIRequest struct {
	Messages []AIMessage `json:"messages"`
	OpenAISettings
}

// OpenAIRequest defines the request structure specific to OpenAI.
// https://platform.openai.com/docs/api-reference/chat/create
type OpenAISettings struct {
	Model               string `json:"model"`
	MaxCompletionTokens *int   `json:"max_completion_tokens,omitempty"`
	ReasoningEffort     string `json:"reasoning_effort,omitempty"` // low, medium, high
	// Number between -2.0 and 2.0. Positive values penalize new tokens based on their existing frequency in the text so far, decreasing the model's likelihood to repeat the same line verbatim.
	FrequencyPenalty *float64              `json:"frequency_penalty,omitempty"`
	N                *int                  `json:"n,omitempty"` // Number of completions to generate for each prompt.
	ResponseFormat   *OpenAIResponseFormat `json:"response_format,omitempty"`
	Stop             *string               `json:"stop,omitempty"`   // Up to 4 sequences where the API will stop generating further tokens.
	Stream           bool                  `json:"stream,omitempty"` // If true, the API will return a response as soon as it becomes available, even if the completion is not finished.
	StreamOptions    struct {
		IncludeUsage bool `json:"include_usage,omitempty"`
	} `json:"stream_options,omitempty"`
	Temperature       int       `json:"temperature,omitempty"` // Number between 0 and 1 that controls randomness of the output.
	TopP              int       `json:"top_p,omitempty"`       // Number between 0 and 1 that controls the cumulative probability of the output.
	Tools             []AITools `json:"tools,omitempty"`
	ToolChoice        string    `json:"tool_choice,omitempty"` // Auto but can be used to force to used a tools
	ParallelToolCalls bool      `json:"parallel_tool_calls"`
}

func (s *OpenAISettings) SetMaxTokens(maxTokens int) {
	s.MaxCompletionTokens = &maxTokens
}

func (s *OpenAISettings) SetTools(tools []AITools) {
	s.Tools = tools
	s.ParallelToolCalls = true
}

func (s *OpenAISettings) SetStream(stream bool) {
	s.Stream = stream
	s.StreamOptions.IncludeUsage = true
}

func (s *OpenAISettings) SetModel(model string) {
	s.Model = model
}

type AIToolCall struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Function AIFunctionCall `json:"function"`
}

type OpenAIChoice struct {
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason,omitempty"`
}

type OpenAIMessage struct {
	Role       string       `json:"role"`
	Content    string       `json:"content"`
	ToolCalls  []AIToolCall `json:"tool_calls,omitempty"`
	ToolCallId string       `json:"tool_call_id,omitempty"`
}

// OpenAIResponse defines the response structure specific to OpenAI.
type OpenAIResponse struct {
	ID      string         `json:"id"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
}

type OpenAIUsage struct {
	CompletionTokens        int `json:"completion_tokens"`
	PromptTokens            int `json:"prompt_tokens"`
	TotalTokens             int `json:"total_tokens"`
	CompletionTokensDetails struct {
		AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
		AudioTokens              int `json:"audio_tokens"`
		ReasoningTokens          int `json:"reasoning_tokens"`
		RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
	}
	PromptTokensDetails struct {
		CachedTokens int `json:"cached_tokens"`
		AutdioTokens int `json:"audio_tokens"`
	} `json:"prompt_tokens_details"`
}

func (u OpenAIUsage) GetInputTokens() int {
	return u.PromptTokens
}

func (u OpenAIUsage) GetOutputTokens() int {
	return u.CompletionTokens
}

func (u OpenAIUsage) GetCachedTokens() int {
	return u.PromptTokensDetails.CachedTokens
}

func (r OpenAIResponse) GetUsage() AIUsage {
	return r.Usage
}

func (r OpenAIResponse) GetContent() string {
	return r.Choices[0].Message.Content
}

func (r OpenAIResponse) GetFinishReason() string {
	return r.Choices[0].FinishReason
}

func (r OpenAIResponse) GetChoice() AIChoice {
	return r.Choices[0]
}

func (r OpenAIChoice) GetMessage() AIMessage {
	return r.Message
}

func (r OpenAIChoice) GetFinishReason() string {
	return r.FinishReason
}

func (m OpenAIMessage) GetRole() string {
	return m.Role
}

func (m OpenAIMessage) GetContent() string {
	return m.Content
}

func (m OpenAIMessage) GetToolCalls() []AIToolCall {
	return m.ToolCalls
}

// GenerateResponse sends a request to OpenAI's API and parses the response.
func (p *OpenAIProvider) GenerateResponse(messages []AIMessage) (AIResponse, error) {
	headers := map[string]string{}
	p.setAuthorizationHeader(headers)

	req := OpenAIRequest{
		Messages:       messages,
		OpenAISettings: p.settings,
	}

	var resp OpenAIResponse
	err := p.makeRequest("POST", p.baseUrl, headers, req, &resp)
	if err != nil {
		return nil, err
	}

	fmt.Println(resp)
	return resp, nil
}
