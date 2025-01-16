package base

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/requestoption"
	"github.com/y0ug/ai-helper/pkg/llmclient/stream"
)

type ChatService interface {
	New(ctx context.Context, params any, opts ...requestoption.RequestOption) (any, error)

	NewStreaming(
		ctx context.Context,
		params any,
		opts ...requestoption.RequestOption,
	) stream.Streamer[common.LLMProvider]
}
