package base

import (
	"context"
	"fmt"
	"net/http"

	"github.com/y0ug/ai-helper/pkg/llmclient/http/requestconfig"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/requestoption"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/streaming"
)

// BaseChatService is a generic base implementation of ChatService.
type BaseChatService[Params any, Response any, Chunk any] struct {
	Options  []requestoption.RequestOption
	NewError requestconfig.NewAPIError
	Endpoint string
}

// New creates a new chat completion.
func (svc *BaseChatService[Params, Response, Chunk]) New(
	ctx context.Context,
	params Params,
	opts ...requestoption.RequestOption,
) (Response, error) {
	var res Response
	combinedOpts := append(svc.Options, opts...)
	path := svc.Endpoint

	err := requestconfig.ExecuteNewRequest(
		ctx,
		http.MethodPost,
		path,
		params,
		&res,
		svc.NewError,
		combinedOpts...,
	)
	return res, err
}

// NewStreaming creates a new streaming chat completion.
func (svc *BaseChatService[Params, Response, Chunk]) NewStreaming(
	ctx context.Context,
	params Params,
	opts ...requestoption.RequestOption,
) (streaming.Streamer[Chunk], error) {
	combinedOpts := append(svc.Options, opts...)
	combinedOpts = append(
		[]requestoption.RequestOption{
			requestoption.WithJSONSet("stream", true),
			requestoption.WithJSONSet("stream_options", struct {
				IncludeUsage bool `json:"include_usage"`
			}{IncludeUsage: true}),
		},
		combinedOpts...)
	path := svc.Endpoint

	var raw *http.Response
	err := requestconfig.ExecuteNewRequest(
		ctx,
		http.MethodPost,
		path,
		params,
		&raw,
		svc.NewError,
		combinedOpts...,
	)
	if err != nil {
		return nil, fmt.Errorf("error executing new request streaming: %w", err)
	}
	return streaming.NewStream(
		streaming.NewDecoderSSE(raw),
		streaming.NewDefaultStreamHandler[Chunk](),
	), nil
}
