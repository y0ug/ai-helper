package ai

type ToolCall struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Args string `json:"arguments"`
	Type string `json:"type"`
}

type Response struct {
	Content        string     `json:"content"`
	Error          error      `json:"-"`
	RequiresAction bool       `json:"requires_action"`
	ToolCalls      []ToolCall `json:"tool_calls,omitempty"`
	InputTokens    int        `json:"input_tokens"`
	OutputTokens   int        `json:"output_tokens"`
	CachedTokens   int        `json:"cached_tokens"`
	Cost           *float64   `json:"cost,omitempty"`
}

type Provider interface {
	SetMaxTokens(maxTokens int)
	SetTools(tool []AITools) 
	GenerateResponse(messages []Message) (Response, error)
}
