package coder

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/coder/prompts"
	"github.com/y0ug/ai-helper/internal/coder/settings"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
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
) ([]prompts.Message, error) {
	messagesLLM := c.convertToLLMMessages(messages)

	chatParams := chat.NewChatParams(
		chat.WithMaxTokens(c.settings.GetMaxOutputToken()),
		chat.WithModel(c.settings.GetModelName()),
		chat.WithMessages(messagesLLM...))

	if c.processor != nil && c.processor.HasWriter() {
		return c.handleStreamingResponse(ctx, chatParams)
	}

	return c.handleNonStreamingResponse(ctx, chatParams)
}

func (c *LLMClient) convertToLLMMessages(messages *ChatChunks) []*chat.ChatMessage {
	var messagesLLM []*chat.ChatMessage

	// Convert system messages
	for _, m := range messages.System {
		messagesLLM = append(messagesLLM,
			chat.NewMessage("system", chat.NewTextContent(m.Content)))
	}

	// Convert all other messages
	for _, m := range messages.AllMessages() {
		messagesLLM = append(messagesLLM,
			chat.NewMessage(m.Role, chat.NewTextContent(m.Content)))
	}

	return messagesLLM
}

func (c *LLMClient) handleNonStreamingResponse(
	ctx context.Context,
	chatParams *chat.ChatParams,
) ([]prompts.Message, error) {
	resp, err := c.client.Send(ctx, *chatParams)
	if err != nil {
		return nil, fmt.Errorf("error chatting: %w", err)
	}

	return c.processResponse(resp)
}

func (c *LLMClient) handleStreamingResponse(
	ctx context.Context,
	chatParams *chat.ChatParams,
) ([]prompts.Message, error) {
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

func (c *LLMClient) processResponse(resp *chat.ChatResponse) ([]prompts.Message, error) {
	msgParams := resp.ToMessageParams()
	c.logger.Debug("msg", "role", msgParams.Role, "content", msgParams.Content)

	return []prompts.Message{{
		Role:    msgParams.Role,
		Content: msgParams.Content[0].String(),
	}}, nil
}
