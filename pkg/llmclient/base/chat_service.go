package base

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/http/requestoption"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/streaming"
)

type ChatService[Params any, Response any, Chunk any] interface {
	New(ctx context.Context, params Params, opts ...requestoption.RequestOption) (Response, error)

	NewStreaming(
		ctx context.Context,
		params Params,
		opts ...requestoption.RequestOption,
	) streaming.Streamer[Chunk]
}
