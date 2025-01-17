package types

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/http/streaming"
)

type LLMProvider interface {
	// For a single-turn request
	Send(ctx context.Context, params ChatMessageNewParams) (*ChatMessage, error)

	// For streaming support
	Stream(
		ctx context.Context,
		params ChatMessageNewParams,
	) (streaming.Streamer[EventStream], error)
}
