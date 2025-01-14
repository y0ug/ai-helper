package base

import (
	"context"
	"net/http"

	"github.com/y0ug/ai-helper/pkg/llmclient/v2/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestconfig"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestoption"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/ssestream"
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
) common.Streamer[Chunk] {
	combinedOpts := append(svc.Options, opts...)
	combinedOpts = append(
		[]requestoption.RequestOption{requestoption.WithJSONSet("stream", true)},
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
		return ssestream.NewBaseStream[Chunk](nil, err)
	}
	return ssestream.NewBaseStream[Chunk](ssestream.NewDecoder(raw), nil)
}
