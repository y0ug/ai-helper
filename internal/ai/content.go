package ai

import "fmt"

// ContentType enumerates possible content types we handle
type ContentType string

const (
	ContentTypeText       ContentType = "text"
	ContentTypeToolUse    ContentType = "tool_use"
	ContentTypeToolResult ContentType = "tool_result"
)

// AIContent holds text, tool calls, or other specialized content
type AIContent struct {
	Type ContentType `json:"type"`

	// Relevant for text content
	Text string `json:"text,omitempty"`

	// Relevant for tool usage calls (like "function calls")
	ToolID     string                 `json:"tool_id,omitempty"`     // Unique identifier for this tool call
	ToolName   string                 `json:"tool_name,omitempty"`   // Name of the tool to call
	Arguments  map[string]interface{} `json:"arguments,omitempty"`   // Arguments to pass to the tool

	// Relevant for tool results
	ToolUseID string `json:"tool_use_id,omitempty"` // ID of the tool call this result is for
	Result    string `json:"result,omitempty"`       // Result returned from the tool
}

// NewTextContent creates a text content message
func NewTextContent(text string) AIContent {
	return AIContent{
		Type: ContentTypeText,
		Text: text,
	}
}

// NewToolUseContent creates a tool use content message
func NewToolUseContent(id, name string, args map[string]interface{}) AIContent {
	return AIContent{
		Type:      ContentTypeToolUse,
		ToolID:    id,
		ToolName:  name,
		Arguments: args,
	}
}

// NewToolResultContent creates a tool result content message
func NewToolResultContent(toolUseID, result string) AIContent {
	return AIContent{
		Type:      ContentTypeToolResult,
		ToolUseID: toolUseID,
		Result:    result,
	}
}

// GetType returns the content type
func (c AIContent) GetType() string {
	return string(c.Type)
}

// String returns a human-readable string (for debugging/logging)
func (c AIContent) String() string {
	switch c.Type {
	case ContentTypeText:
		return c.Text
	case ContentTypeToolUse:
		args, _ := json.Marshal(c.Arguments)
		return fmt.Sprintf("%s:%s => %s", c.ToolID, c.ToolName, string(args))
	case ContentTypeToolResult:
		return fmt.Sprintf("Result[%s]: %s", c.ToolUseID, c.Result)
	default:
		return fmt.Sprintf("unknown content type: %s", c.Type)
	}
}

// Raw returns the entire struct as a generic interface
func (c AIContent) Raw() interface{} {
	return c
}
