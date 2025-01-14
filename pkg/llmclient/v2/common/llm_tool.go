package common

type LLMTool struct {
	Description *string     `json:"description,omitempty"`
	InputSchema interface{} `json:"input_schema,omitempty"`
	Name        string      `json:"name"`
}
