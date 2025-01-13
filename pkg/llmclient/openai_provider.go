package llmclient

import (
	"encoding/json"
	"net/http"
)

// OpenAIProvider implements the Provider interface for OpenAI's API.
type OpenAIProvider struct {
	BaseProvider
	settings *OpenAISettings
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
	logger *zerolog.Logger,
) (*OpenAIProvider, error) {
	return &OpenAIProvider{
		BaseProvider: *NewBaseProvider(model, apiKey, client, apiUrl, logger),
		settings:     &OpenAISettings{Model: model.Name},
	}, nil
}

func (p *OpenAIProvider) Settings() AIModelSettings {
	return p.settings
}

type OpenAIRequest struct {
	Messages []OpenAIMessage `json:"messages"`
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
	StreamOptions    *struct {
		IncludeUsage bool `json:"include_usage,omitempty"`
	} `json:"stream_options,omitempty"`
	Temperature int          `json:"temperature,omitempty"` // Number between 0 and 1 that controls randomness of the output.
	TopP        int          `json:"top_p,omitempty"`       // Number between 0 and 1 that controls the cumulative probability of the output.
	Tools       []OpenAITool `json:"tools,omitempty"`
	ToolChoice  interface{}  `json:"tool_choice,omitempty"` // Auto but can be used to force to used a tools
	// ParallelToolCalls bool      `json:"parallel_tool_calls"`
}

func (s *OpenAISettings) SetMaxTokens(maxTokens int) {
	s.MaxCompletionTokens = &maxTokens
}

func (s *OpenAISettings) SetTools(tools []AITools) {
	toolList := make([]OpenAITool, 0)
	for _, t := range tools {
		toolList = append(toolList, AIToolToOpenAITool(t))
	}

	s.Tools = toolList
	// s.ParallelToolCalls = true
}

func (s *OpenAISettings) SetStream(stream bool) {
	s.Stream = stream
	s.StreamOptions.IncludeUsage = true
}

func (s *OpenAISettings) SetModel(model string) {
	s.Model = model
}

// Description of a tool_calls message request from assistant
type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function OpenAIFunctionCall `json:"function"`
}

func toolCallToAIContent(t OpenAIToolCall) *AIContent {
	var args map[string]interface{}
	_ = json.Unmarshal([]byte(t.Function.Arguments), &args)
	return NewToolUseContent(t.ID, t.Function.Name, args)
}

// Description of a tool function
type OpenAIToolFunction struct {
	Name        string      `json:"name"`
	Description *string     `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters"`
}

type OpenAITool struct {
	Type     string             `json:"type"`
	Function OpenAIToolFunction `json:"function"`
}

func AIToolToOpenAITool(tool AITools) OpenAITool {
	var desc *string
	if tool.Description != nil {
		descCopy := *tool.Description
		desc = &descCopy
		if len(*desc) > 512 {
			foo := descCopy[:512]
			desc = &foo
		}
	}
	aiTool := OpenAITool{
		Type: "function",
		Function: OpenAIToolFunction{
			Name:        tool.Name,
			Description: desc,
			Parameters:  tool.InputSchema,
		},
	}
	return aiTool
}

type OpenAIChoice struct {
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason,omitempty"`
}

type OpenAIMessage struct {
	Role       string           `json:"role"`
	Refusal    string           `json:"refusal,omitempty"`
	Name       string           `json:"name,omitempty"`
	Audio      interface{}      `json:"audio,omitempty"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	Content    string           `json:"content,omitempty"`
	ToolCallId string           `json:"tool_call_id,omitempty"`
}

// OpenAIResponse defines the response structure specific to OpenAI.
type OpenAIResponse struct {
	ID      string         `json:"id"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   *OpenAIUsage   `json:"usage"`
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
	Cost float64 `json:"cost,omitempty"`
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

func (u OpenAIUsage) GetCost() float64 {
	return u.Cost
}

func (u *OpenAIUsage) SetCost(cost float64) {
	u.Cost = cost
}

func (r OpenAIResponse) GetUsage() AIUsage {
	return r.Usage
}

func (r OpenAIResponse) GetChoice() AIChoice {
	if len(r.Choices) == 0 {
		return nil
	}
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

func (m OpenAIMessage) GetContent() *AIContent {
	if len(m.ToolCalls) > 0 {
		return toolCallToAIContent(m.ToolCalls[0])
	}
	return NewTextContent(m.Content)
}

func (m OpenAIMessage) GetContents() []*AIContent {
	if len(m.ToolCalls) > 0 {
		contents := make([]*AIContent, len(m.ToolCalls))
		for i, tc := range m.ToolCalls {
			contents[i] = toolCallToAIContent(tc)
		}
		return contents
	}
	return []*AIContent{NewTextContent(m.Content)}
}

func AIMessageToOpenAIMessage(m []AIMessage) []OpenAIMessage {
	userMessages := make([]OpenAIMessage, 0)
	for _, msg := range m {
		switch msg := msg.(type) {
		case OpenAIMessage:
			userMessages = append(userMessages, msg)
		default:
			content := msg.GetContent()
			if content.Type == ContentTypeToolResult {
				userMessages = append(userMessages, OpenAIMessage{
					Role:       "tool",
					Content:    content.Content,
					ToolCallId: content.ToolUseID,
				})
			} else {
				userMessages = append(userMessages, OpenAIMessage{
					Role:    msg.GetRole(),
					Content: content.String(),
				})
			}
		}
	}
	return userMessages
}

// GenerateResponse sends a request to OpenAI's API and parses the response.
func (p *OpenAIProvider) GenerateResponse(messages []AIMessage) (AIResponse, error) {
	headers := map[string]string{}
	p.setAuthorizationHeader(headers)

	req := OpenAIRequest{
		Messages:       AIMessageToOpenAIMessage(messages),
		OpenAISettings: *p.settings,
	}
	// for _, msg := range req.Messages {
	// 	data, _ := json.Marshal(msg)
	// 	fmt.Fprintf(os.Stderr, "msg: %T %v\n", msg, string(data))
	// }

	var resp OpenAIResponse
	err := p.makeRequest("POST", p.baseUrl, headers, req, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
