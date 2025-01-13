package llmclient

import (
	"fmt"
	"net/http"
	"os"
)

// AnthropicProvider implements the Provider interface for Anthropic's API.
type AnthropicProvider struct {
	BaseProvider
	settings *AnthropicSettings
}

// NewAnthropicProvider creates a new instance of AnthropicProvider.
func NewAnthropicProvider(
	model *Model,
	apiKey string,
	client *http.Client,
	apiUrl string,
	logger *zerolog.Logger,
) (*AnthropicProvider, error) {
	return &AnthropicProvider{
		BaseProvider: *NewBaseProvider(model, apiKey, client, apiUrl, logger),
		settings:     &AnthropicSettings{Model: model.Name},
	}, nil
}

func (p *AnthropicProvider) Settings() AIModelSettings {
	return p.settings
}

type AnthropicSettings struct {
	MaxTokens     int      `json:"max_tokens,omitempty"`
	Model         string   `json:"model"`
	StopSequences []string `json:"stop_sequences,omitempty"`
	Stream        bool     `json:"stream,omitempty"`
	System        string   `json:"system,omitempty"`

	Temperature int         `json:"temperature,omitempty"` // Number between 0 and 1 that controls randomness of the output.
	Tools       []AITools   `json:"tools,omitempty"`
	ToolChoice  interface{} `json:"tool_choice,omitempty"` // Auto but can be used to force to used a tools
}

func (s *AnthropicSettings) SetMaxTokens(maxTokens int) {
	s.MaxTokens = maxTokens
}

func (s *AnthropicSettings) SetTools(tools []AITools) {
	s.Tools = tools
	// s.ParallelToolCalls = true
}

func (s *AnthropicSettings) SetStream(stream bool) {
	s.Stream = stream
}

func (s *AnthropicSettings) SetModel(model string) {
	s.Model = model
}

type AnthropicResponse struct {
	ID           string          `json:"id"`
	Content      []*AIContent    `json:"content"`
	Role         string          `json:"role"` // Always "assistant"
	StopReason   string          `json:"stop_reason,omitempty"`
	StopSequence string          `json:"stop_sequence,omitempty"`
	Type         string          `json:"type"` // Always "message"
	Usage        *AnthropicUsage `json:"usage"`
	Model        string          `json:"model"`
}

func (r AnthropicResponse) GetMessage() AIMessage {
	return BaseMessage{
		Role:    r.Role,
		Content: r.Content,
	}
}

// AnthropicResponse Implement AIResponse interface
func (r AnthropicResponse) GetChoice() AIChoice {
	return r
}

func (r AnthropicResponse) GetFinishReason() string {
	if r.StopReason == "tool_use" {
		return "tool_calls"
	}
	return r.StopReason
}

func (r AnthropicResponse) GetUsage() AIUsage {
	return r.Usage
}

type AnthropicRequest struct {
	Messages []BaseMessage `json:"messages"`
	AnthropicSettings
}

// Implement AIUsage interface
type AnthropicUsage struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Cost         float64 `json:"cost,omitempty"` // No  in API we compute at the end
}

func (r AnthropicUsage) GetInputTokens() int {
	return r.InputTokens
}

func (r AnthropicUsage) GetOutputTokens() int {
	return r.OutputTokens
}

func (r AnthropicUsage) GetCachedTokens() int {
	return 0
}

func (r AnthropicUsage) GetCost() float64 {
	return r.Cost
}

func (r *AnthropicUsage) SetCost(cost float64) {
	r.Cost = cost
}

// GenerateResponse sends a request to Anthropic's API and parses the response.
func (p *AnthropicProvider) GenerateResponse(messages []AIMessage) (AIResponse, error) {
	if p.settings.MaxTokens == 0 {
		p.settings.MaxTokens = 4096
	}
	var userMessages []BaseMessage

	for _, msg := range messages {
		if msg.GetRole() == "system" {
			p.settings.System = msg.GetContent().String()
			continue
		}
		switch m := msg.(type) {
		case BaseMessage:
			userMessages = append(userMessages, m)
		default:
			userMessages = append(userMessages, BaseMessage{
				Role:    m.GetRole(),
				Content: m.GetContents(),
			})
			fmt.Fprintf(os.Stderr, "unsupported msg: %T %v\n", msg, msg)
		}
	}

	req := AnthropicRequest{
		AnthropicSettings: *p.settings,
		Messages:          userMessages,
	}
	// for _, msg := range req.Messages {
	// 	data, _ := json.Marshal(msg)
	// 	fmt.Fprintf(os.Stderr, "msg: %T %v\n", msg, string(data))
	// }

	var apiResp AnthropicResponse

	headers := map[string]string{
		"anthropic-version": "2023-06-01",
		"x-api-key":         p.apiKey,
	}

	err := p.makeRequest("POST", p.baseUrl, headers, req, &apiResp)
	if err != nil {
		return nil, err
	}

	return apiResp, nil
}
