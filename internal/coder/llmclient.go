package coder

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/coder/settings"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type LLMClient struct {
	client    chat.Provider
	settings  *settings.CoderSettings
	logger    *slog.Logger
	processor *StreamProcessor
}

func NewLLMClient(
	client chat.Provider,
	settings *settings.CoderSettings,
	logger *slog.Logger,
	processor *StreamProcessor,
) *LLMClient {
	return &LLMClient{
		client:    client,
		settings:  settings,
		logger:    logger,
		processor: processor,
	}
}

func (c *LLMClient) SendMessages(
	ctx context.Context,
	messages *ChatChunks,
	tools []chat.Tool,
) (*chat.ChatResponse, error) {
	messagesLLM := messages.AllMessages()

	chatParams := chat.NewChatParams(
		chat.WithMaxTokens(c.settings.GetMaxOutputToken()),
		chat.WithModel(c.settings.GetModelName()),
		chat.WithMessages(messagesLLM...),
		chat.WithTools(tools...))

	fn := c.handleNonStreamingResponse

	if c.processor != nil && c.processor.HasWriter() {
		fn = c.handleStreamingResponse
	}

	resp, err := fn(ctx, chatParams)
	if err != nil {
		return nil, err
	}

	return resp, err
}

func (c *LLMClient) handleNonStreamingResponse(
	ctx context.Context,
	chatParams *chat.ChatParams,
) (*chat.ChatResponse, error) {
	resp, err := c.client.Send(ctx, *chatParams)
	if err != nil {
		return nil, fmt.Errorf("error chatting: %w", err)
	}

	return c.processResponse(resp)
}

func (c *LLMClient) handleStreamingResponse(
	ctx context.Context,
	chatParams *chat.ChatParams,
) (*chat.ChatResponse, error) {
	respChan, err := c.client.Stream(ctx, *chatParams)
	if err != nil {
		return nil, fmt.Errorf("error streaming response: %w", err)
	}

	resp, err := c.processor.ProcessStream(ctx, respChan)
	if err != nil {
		return nil, fmt.Errorf("error processing stream: %w", err)
	}

	return c.processResponse(resp)
}

func (c *LLMClient) processResponse(resp *chat.ChatResponse) (*chat.ChatResponse, error) {
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
