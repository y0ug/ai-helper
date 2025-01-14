package base

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestoption"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/ssestream"
)

type ChatService interface {
	New(ctx context.Context, params any, opts ...requestoption.RequestOption) (any, error)

	NewStreaming(
		ctx context.Context,
		params any,
		opts ...requestoption.RequestOption,
	) ssestream.Streamer[any]
}
