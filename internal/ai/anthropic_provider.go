package ai

import (
	"encoding/json"
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
) (*AnthropicProvider, error) {
	return &AnthropicProvider{
		BaseProvider: *NewBaseProvider(model, apiKey, client, apiUrl),
		settings:     &AnthropicSettings{Model: model.Name},
	}, nil
}

func (p *AnthropicProvider) Settings() AIModelSettings {
	return p.settings
}

type AnthropicTool struct {
	Name        string      `json:"name"`
	Description *string     `json:"description",omitempty`
	InputSchema interface{} `json:"input_schema"`
}

func (v *AITools) ToAntropicTool() *AnthropicTool {
	return &AnthropicTool{
		Name:        v.Function.Name,
		Description: v.Function.Description,
		InputSchema: v.Function.Parameters,
	}
}

type AnthropicSettings struct {
	MaxTokens     int      `json:"max_tokens,omitempty"`
	Model         string   `json:"model"`
	StopSequences []string `json:"stop_sequences,omitempty"`
	Stream        bool     `json:"stream,omitempty"`
	System        string   `json:"system,omitempty"`

	Temperature int             `json:"temperature,omitempty"` // Number between 0 and 1 that controls randomness of the output.
	Tools       []AnthropicTool `json:"tools,omitempty"`
	ToolChoice  interface{}     `json:"tool_choice,omitempty"` // Auto but can be used to force to used a tools
}

func (s *AnthropicSettings) SetMaxTokens(maxTokens int) {
	s.MaxTokens = maxTokens
}

func (s *AnthropicSettings) SetTools(tools []AITools) {
	s.Tools = []AnthropicTool{}
	for _, t := range tools {
		s.Tools = append(s.Tools, *t.ToAntropicTool())
	}
	// s.ParallelToolCalls = true
}

func (s *AnthropicSettings) SetStream(stream bool) {
	s.Stream = stream
}

func (s *AnthropicSettings) SetModel(model string) {
	s.Model = model
}

// Should implement AIMessage interface
type AnthropicMessage []*AnthropicContent

func (m AnthropicMessage) GetRole() string {
	return "assistant"
}

func (m AnthropicMessage) GetContent() AIContent {
	return m[0]
}

func (m AnthropicMessage) GetContents() []AIContent {
	contents := make([]AIContent, 0)
	for _, c := range m {
		if c != nil && c.AIContent != nil {
			contents = append(contents, c.AIContent)
		}
	}
	return contents
}

// func (m AnthropicMessage) GetToolCalls() []AIToolCall {
// 	AIToolCalls := []AIToolCall{}
// 	for _, cw := range m {
// 		switch c := cw.(type) {
// 		case AnthropicContentToolUse:
// 			args, _ := json.Marshal(c.Input)
// 			AIToolCalls = append(AIToolCalls, AIToolCall{
// 				ID:   c.ID,
// 				Type: "function",
// 				Function: AIFunctionCall{
// 					Name:      c.Name,
// 					Arguments: string(args),
// 				},
// 			})
// 		}
// 	}
// 	return AIToolCalls
// }

// Antropic response compare to openAPI
// Choice[0] is the array Content field of the response
// A response can have more then one content

type AnthropicResponse struct {
	ID           string           `json:"id"`
	Content      AnthropicMessage `json:"content"`
	Role         string           `json:"role"` // Always "assistant"
	StopReason   string           `json:"stop_reason,omitempty"`
	StopSequence string           `json:"stop_sequence,omitempty"`
	Type         string           `json:"type"` // Always "message"
	Usage        AnthropicUsage   `json:"usage"`
	Model        string           `json:"model"`
}

func (r AnthropicResponse) GetMessage() AIMessage {
	return AnthropicMessageRequest{
		Role:    r.Role,
		Content: r.Content.GetContents(),
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
	Messages []AnthropicMessageRequest `json:"messages"`
	AnthropicSettings
}

// Implement AIUsage interface
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
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

// AnthropicContent , Message and  Content
// if we compare to
type AnthropicContent struct {
	AIContent
}

// TextContent represents the "text" type content
type AnthropicContentText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (t AnthropicContentText) GetType() string {
	return t.Type
}

func (t AnthropicContentText) String() string {
	return t.Text
}

func (t AnthropicContentText) Raw() interface{} {
	return t
}

// ToolUseContent represents the "tool_use" type content
type AnthropicContentToolUse struct {
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

func (t AnthropicContentToolUse) GetType() string {
	return t.Type
}

func (t AnthropicContentToolUse) String() string {
	return fmt.Sprintf("%s:%s: %v", t.ID, t.Name, t.Input)
}

func (t AnthropicContentToolUse) Raw() interface{} {
	return t
}

func (t AnthropicContentToolUse) GetID() string {
	return t.ID
}

func (t AnthropicContentToolUse) GetName() string {
	return t.Name
}

func (t AnthropicContentToolUse) GetCallType() string {
	return "function"
}

func (t AnthropicContentToolUse) GetArguments() string {
	args, _ := json.Marshal(t.Input)
	return string(args)
}

func (cw *AnthropicContent) UnmarshalJSON(data []byte) error {
	// Temporary struct to get the type
	var temp struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Based on the type, unmarshal into the appropriate struct
	switch temp.Type {
	case "text":
		var tc AnthropicContentText
		if err := json.Unmarshal(data, &tc); err != nil {
			return err
		}
		cw.AIContent = tc
	case "tool_use":
		var tuc AnthropicContentToolUse
		if err := json.Unmarshal(data, &tuc); err != nil {
			return err
		}
		cw.AIContent = tuc
	// Add more cases for other content types
	default:
		return fmt.Errorf("unknown content type: %s", temp.Type)
	}

	return nil
}

type AnthropicContentToolResult struct {
	Type      string `json:"type"` // tool_result
	ToolUseId string `json:"tool_use_id"`
	Content   string `json:"content"`
}

func (t AnthropicContentToolResult) GetType() string {
	return t.Type
}

func (t AnthropicContentToolResult) String() string {
	return t.Content
}

func (t AnthropicContentToolResult) Raw() interface{} {
	return t
}

// OpenAIResponse defines the response structure specific to OpenAI.

type AnthropicMessageRequest struct {
	Role    string      `json:"role"`
	Content []AIContent `json:"content"`
}

func (m AnthropicMessageRequest) GetRole() string {
	return m.Role
}

func (m AnthropicMessageRequest) GetContent() AIContent {
	return m.Content[0]
}

func (m AnthropicMessageRequest) GetContents() []AIContent {
	return m.Content
}

// GenerateResponse sends a request to Anthropic's API and parses the response.
func (p *AnthropicProvider) GenerateResponse(messages []AIMessage) (AIResponse, error) {
	if p.settings.MaxTokens == 0 {
		p.settings.MaxTokens = 4096
	}
	var userMessages []AnthropicMessageRequest

	for _, msg := range messages {
		if msg.GetRole() == "system" {
			p.settings.System = msg.GetContent().String()
			continue
		}
		userMessages = append(userMessages, AnthropicMessageRequest{
			Role:    msg.GetRole(),
			Content: msg.GetContents(),
		})
	}

	req := AnthropicRequest{
		AnthropicSettings: *p.settings,
		Messages:          userMessages,
	}
	for _, msg := range req.Messages {
		data, _ := json.Marshal(msg)
		fmt.Fprintf(os.Stderr, "msg: %T %v\n", msg, string(data))
	}

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
