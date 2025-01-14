package chat

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient"
)

// Message represents a chat message with role and content
type Message struct {
	Role    string                 `json:"role"`
	Content []*llmclient.AIContent `json:"content"`
}

// Usage tracks token consumption and costs
type Usage struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Cost         float64 `json:"cost,omitempty"`
}

// Response contains the complete chat response
type Response struct {
	ID      string    `json:"id"`
	Message Message   `json:"message"`
	Usage   Usage     `json:"usage"`
	Model   string    `json:"model"`
	Raw     any      `json:"raw,omitempty"` // Provider-specific raw response
}

// StreamEvent represents a streaming response event
type StreamEvent struct {
	Type    string           `json:"type"`
	Message Message          `json:"message,omitempty"`
	Delta   Message          `json:"delta,omitempty"`
	Usage   Usage           `json:"usage,omitempty"`
	Done    bool            `json:"done"`
	Raw     any            `json:"raw,omitempty"` // Provider-specific raw event
}

// Config contains common configuration options
type Config struct {
	Model         string
	MaxTokens     int
	Temperature   float64
	Tools         []llmclient.AITools
	ToolChoice    any
	StopSequences []string
	System        string
}

// Provider defines the interface for chat providers
type Provider interface {
	// Send sends a message and returns the response
	Send(ctx context.Context, messages []Message, cfg *Config) (*Response, error)

	// Stream sends a message and returns a channel of stream events
	Stream(ctx context.Context, messages []Message, cfg *Config) (<-chan StreamEvent, error)
}
