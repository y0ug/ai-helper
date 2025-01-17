package base

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/request/requestoption"
	"github.com/y0ug/ai-helper/pkg/llmclient/stream"
)

type ChatService[Params any, Response any, Chunk any] interface {
	New(ctx context.Context, params Params, opts ...requestoption.RequestOption) (Response, error)

	NewStreaming(
		ctx context.Context,
		params Params,
		opts ...requestoption.RequestOption,
	) stream.Streamer[Chunk]
}
