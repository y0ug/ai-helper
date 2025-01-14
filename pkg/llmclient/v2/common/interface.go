package common

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/v2/ssestream"
)

type LLMClient interface {
	// For a single-turn request
	Send(ctx context.Context, params BaseChatMessageNewParams) (BaseChatMessage, error)

	// For streaming support
	Stream(
		ctx context.Context,
		params BaseChatMessageNewParams,
	) (ssestream.Streamer[LLMStreamEvent], error)
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

type LLMStreamEvent interface{}

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
