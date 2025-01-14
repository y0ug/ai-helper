package common

type BaseChatMessageNewParams struct {
	Model       string
	MaxTokens   int
	Temperature float64
	// ... stop sequences, etc.
	Messages   []*BaseChatMessageParams
	Stream     bool
	Tools      []LLMTool
	ToolChoice interface{}
	// etc...
}

type BaseChatMessage struct {
	ID     string                  `json:"id,omitempty"`
	Choice []BaseChatMessageChoice `json:"choice,omitempty"`
	Usage  *BaseChatMessageUsage   `json:"usage,omitempty"`
	Model  string                  `json:"model,omitempty"`
}

type BaseChatMessageChoice struct {
	Role         string       `json:"role,omitempty"` // Always "assistant"
	Content      []*AIContent `json:"content,omitempty"`
	FinishReason string       `json:"finish_reason,omitempty"`
}
type BaseChatMessageUsage struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Cost         float64 `json:"cost,omitempty"` // No  in API we compute at the end
}

type BaseChatMessageParams struct {
	Role    string       `json:"role"`
	Content []*AIContent `json:"content"`
}

func (m BaseChatMessageParams) GetRole() string {
	return m.Role
}

func (m BaseChatMessageParams) GetContents() []*AIContent {
	return m.Content
}

func (m BaseChatMessageParams) GetContent() *AIContent {
	if len(m.Content) != 0 {
		return m.Content[0]
	}
	return nil
}

func NewBaseMessage(role string, content ...*AIContent) BaseChatMessageParams {
	return BaseChatMessageParams{
		Role:    role,
		Content: content,
	}
}

func NewBaseMessageText(role string, text string) BaseChatMessageParams {
	return BaseChatMessageParams{
		Role: role,
		Content: []*AIContent{
			NewTextContent(text),
		},
	}
}
