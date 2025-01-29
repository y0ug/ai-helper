package llm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/assistant/settings"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type Completion func(ctx context.Context, messages []*chat.ChatMessage, tools []chat.Tool) (*chat.ChatResponse, error)

type Client struct {
	client   chat.Provider
	settings *settings.CoderSettings
	logger   *slog.Logger
}

func New(
	client chat.Provider,
	settings *settings.CoderSettings,
	logger *slog.Logger,
	metrics MetricsRecorder,
) ChatCompleter {
	c := &Client{
		client:   client,
		settings: settings,
		logger:   logger,
	}

	return WithMiddleware(c, MetricsMiddleware(metrics))
}

func CombineOptions[T any](opts ...func(*T)) []func(*T) {
	return opts
}

func (c *Client) SendMessages(
	ctx context.Context,
	messages []*chat.ChatMessage,
	tools []chat.Tool,
	opts ...func(*Params),
) (*chat.ChatResponse, error) {
	chatOpts := WithChatParams(
		chat.WithMaxTokens(c.settings.GetMaxOutputToken()),
		chat.WithModel(c.settings.GetModelName()),
		chat.WithMessages(messages...),
		chat.WithTools(tools...))

	// Opts can overide the default one
	params := NewParams(append([]func(*Params){
		chatOpts,
		WithStream(false),
	}, opts...)...)

	fn := c.handleNonStreamingResponse

	if params.Stream && params.StreamProcessor != nil && params.StreamProcessor.HasWriter() {
		fn = c.handleStreamingResponse
	}

	resp, err := fn(ctx, params)
	if err != nil {
		return nil, err
	}

	return resp, err
}

func (c *Client) handleNonStreamingResponse(
	ctx context.Context,
	params *Params,
) (*chat.ChatResponse, error) {
	resp, err := c.client.Send(ctx, *params.ChatParams)
	if err != nil {
		return nil, fmt.Errorf("error chatting: %w", err)
	}

	return c.processResponse(resp)
}

func (c *Client) handleStreamingResponse(
	ctx context.Context,
	params *Params,
) (*chat.ChatResponse, error) {
	respChan, err := c.client.Stream(ctx, *params.ChatParams)
	if err != nil {
		return nil, fmt.Errorf("error streaming response: %w", err)
	}

	resp, err := params.StreamProcessor.ProcessStream(ctx, respChan)
	if err != nil {
		return nil, fmt.Errorf("error processing stream: %w", err)
	}

	return c.processResponse(resp)
}

func (c *Client) processResponse(resp *chat.ChatResponse) (*chat.ChatResponse, error) {
	if resp == nil {
		return nil, fmt.Errorf("error processing response, nil response")
	}
	msgParams := resp.ToMessageParams()
	if msgParams == nil {
		return nil, fmt.Errorf("error converting response to message params, no choice")
	}

	c.logger.Debug("msg", "role", msgParams.Role, "content", msgParams.Content)

	c.logger.Info(
		"usage",
		"input_tokens",
		resp.Usage.InputTokens,
		"output_tokens",
		resp.Usage.OutputTokens,
		"input_cached_tokens",
		resp.Usage.InputCachedTokens,
		"input_cache_creation_tokens",
		resp.Usage.InputCacheCreationTokens,
		"output_reasoning_tokens",
		resp.Usage.OutputReasoningTokens,
	)
	return resp, nil
}
