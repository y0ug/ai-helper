package common

import (
	"context"
	"encoding/json"
)

type Streamer[T any] interface {
	Next() bool
	Current() T
	Err() error
	Close() error
}

type LLMStreamEvent struct {
	Provider string          `json:"provider"` // e.g., "anthropic", "openai"
	Type     string          `json:"type"`     // Event type identifier
	Data     json.RawMessage `json:"data"`     // Raw JSON data for provider-specific events
}
type LLMProvider interface {
	// For a single-turn request
	Send(ctx context.Context, params BaseChatMessageNewParams) (*BaseChatMessage, error)

	// For streaming support
	Stream(
		ctx context.Context,
		params BaseChatMessageNewParams,
	) (Streamer[LLMStreamEvent], error)
}

// / type LLMResponse interface {
// 	GetChoices() []LLMChoice
// 	GetUsage() LLMUsage
// }
//
// type LLMUsage interface {
// 	GetInputTokens() int
// 	GetOutputTokens() int
// 	GetCachedTokens() int
// 	GetCost() float64
// 	SetCost(float64)
// }
//
// type LLMChoice interface {
// 	GetContents() []AIContent
// 	GetRole() string
// 	GetFinishReason() string
// }

// func (m *BaseChatMessage) GetChoices() []BaseChatMessageChoice {
// 	return m.Choice
// }
//
// func (c *BaseChatMessageChoice) GetContents() []*AIContent {
// 	return c.Content
// }
//
// func (c *BaseChatMessageChoice) GetRole() string {
// 	return c.Role
// }
//
// func (c *BaseChatMessageChoice) GetFinishReason() string {
// 	return c.FinishReason
// }
