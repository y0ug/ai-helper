package types

type ChatMessageNewParams struct {
	Model       string
	MaxTokens   int
	Temperature float64
	Messages    []*ChatMessageParams
	Stream      bool
	Tools       []Tool
	ToolChoice  string // auto, any, tool
	N           *int   // number of choice
}

type ChatMessage struct {
	ID     string              `json:"id,omitempty"`
	Choice []ChatMessageChoice `json:"choice,omitempty"`
	Usage  *ChatMessageUsage   `json:"usage,omitempty"`
	Model  string              `json:"model,omitempty"`
}

func (cm *ChatMessage) ToMessageParams() *ChatMessageParams {
	return &ChatMessageParams{
		Content: cm.Choice[0].Content,
		Role:    cm.Choice[0].Role,
	}
}

// StopReason is the reason the model stopped generating messages. It can be one of:
// - `"end_turn"`: the model reached a natural stopping point
// - `"max_tokens"`: we exceeded the requested `max_tokens` or the model's maximum
// - `"stop_sequence"`: one of your provided custom `stop_sequences` was generated
// - `"tool_use"`: the model invoked one or more tools
type ChatMessageChoice struct {
	Role       string       `json:"role,omitempty"` // Always "assistant"
	Content    []*AIContent `json:"content,omitempty"`
	StopReason string       `json:"stop_reason,omitempty"`
}

type ChatMessageUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type ChatMessageParams struct {
	Role    string       `json:"role"`
	Content []*AIContent `json:"content"`
}

func (m ChatMessageParams) GetRole() string {
	return m.Role
}

func (m ChatMessageParams) GetContents() []*AIContent {
	return m.Content
}

func (m ChatMessageParams) GetContent() *AIContent {
	if len(m.Content) != 0 {
		return m.Content[0]
	}
	return nil
}

type Tool struct {
	Description *string     `json:"description,omitempty"`
	InputSchema interface{} `json:"input_schema,omitempty"`
	Name        string      `json:"name"`
}
