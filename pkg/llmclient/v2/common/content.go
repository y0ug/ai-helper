package common

import (
	"encoding/json"
	"fmt"
	"log"
)

// ContentType enumerates possible content types we handle
type ContentType string

const (
	// Common Anthropic/OpenAI
	ContentTypeText      ContentType = "text"
	ContentTypeTextDelta ContentType = "text_delta"

	// Anthropic
	ContentTypeToolUse    ContentType = "tool_use"
	ContentTypeToolResult ContentType = "tool_result"
	ContentTypeDocument   ContentType = "document"
	ContentTypeImage      ContentType = "image"

	// OpenAI
	ContentTypeInputAudio ContentType = "input_audio"
)

// AIContent holds text, tool calls, or other specialized content
type AIContent struct {
	Type ContentType `json:"type"`

	// Relevant for text content
	Text string `json:"text,omitempty"`

	// Relevant for tool usage calls (like "function calls")
	ID    string                 `json:"id,omitempty"`    // Unique identifier for this tool call
	Name  string                 `json:"name,omitempty"`  // Name of the tool to call
	Input map[string]interface{} `json:"input,omitempty"` // Arguments to pass to the tool

	// Relevant for tool results
	ToolUseID string        `json:"tool_use_id,omitempty"` // ID of the tool call this result is for
	Content   string        `json:"content,omitempty"`     // Result returned from the tool
	Source    *AIContentSrc `json:"source,omitempty"`      // Source of the content if type document/image
}

type AIContentSrc struct {
	Type      string `json:"type"`       // base64
	MediaType string `json:"media_type"` // "application/pdf" "image/jpeg" etc..
	Data      []byte `json:"data"`
}

// NewTextContent creates a text content message
func NewTextContent(text string) *AIContent {
	return &AIContent{
		Type: ContentTypeText,
		Text: text,
	}
}

func NewSourceContent(sourceType string, mediaType string, data []byte) *AIContent {
	var contentType ContentType
	switch sourceType {
	case "document":
		contentType = ContentTypeDocument
	case "image":
		contentType = ContentTypeImage
	default:
		// TODO: remove this log
		log.Printf("Unknown source type: %s", sourceType)
		contentType = ContentType("contentType")
	}
	return &AIContent{
		Type: contentType,
		Source: &AIContentSrc{
			Type:      "base64",
			MediaType: mediaType,
			Data:      data,
		},
	}
}

// NewToolUseContent creates a tool use content message
func NewToolUseContent(id, name string, args map[string]interface{}) *AIContent {
	return &AIContent{
		Type:  ContentTypeToolUse,
		ID:    id,
		Name:  name,
		Input: args,
	}
}

// NewToolResultContent creates a tool result content message
func NewToolResultContent(toolUseID, content string) *AIContent {
	return &AIContent{
		Type:      ContentTypeToolResult,
		ToolUseID: toolUseID,
		Content:   content,
	}
}

// GetType returns the content type
func (c AIContent) GetType() string {
	return string(c.Type)
}

// String returns a human-readable string (for debugging/logging)
func (c AIContent) String() string {
	switch c.Type {
	case ContentTypeTextDelta:
		return c.Text
	case ContentTypeText:
		return c.Text
	case ContentTypeToolUse:
		args, _ := json.Marshal(c.Input)
		return fmt.Sprintf("%s:%s => %s", c.ID, c.Name, string(args))
	case ContentTypeToolResult:
		return fmt.Sprintf("Result[%s]: %s", c.ToolUseID, c.Content)
	default:
		return fmt.Sprintf("unknown content type: %s", c.Type)
	}
}

// Raw returns the entire struct as a generic interface
func (c AIContent) Raw() interface{} {
	return c
}
