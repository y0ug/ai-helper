package ai

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

    // Relevant for text
    Text string `json:"text,omitempty"`

    // Relevant for tool usage calls
    ToolID     string                 `json:"tool_id,omitempty"`
    ToolName   string                 `json:"tool_name,omitempty"`
    Arguments  map[string]interface{} `json:"arguments,omitempty"`

    // Relevant for tool results
    ToolUseID string `json:"tool_use_id,omitempty"`
    Result    string `json:"result,omitempty"`
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

// String returns a human-readable representation
func (c AIContent) String() string {
    switch c.Type {
    case ContentTypeText:
        return c.Text
    case ContentTypeToolUse:
        return fmt.Sprintf("%s:%s", c.ToolID, c.ToolName)
    case ContentTypeToolResult:
        return c.Result
    default:
        return ""
    }
}

// Raw returns the entire struct
func (c AIContent) Raw() interface{} {
    return c
}
