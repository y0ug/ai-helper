package coder

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/y0ug/ai-helper/internal/coder/prompts"
	"github.com/y0ug/ai-helper/internal/coder/repomanager"
	"github.com/y0ug/ai-helper/internal/coder/responseextractor"
	"github.com/y0ug/ai-helper/internal/coder/settings"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type BaseCoder struct {
	logger        *slog.Logger
	llmClient     *LLMClient
	prompts       prompts.Prompter
	rm            repomanager.RepoManagerInterface
	settings      *settings.CoderSettings
	processor     *MessageProcessor
	history       *ChatHistory
	formatter     *MessageFormatter
	respExtractor []responseextractor.ResponseExtractor
}

func NewBaseCoder(opts CoderOptions) *BaseCoder {
	history := NewChatHistory()
	formatter := NewMessageFormatter(opts.Logger, opts.RepoManager, opts.Prompts, opts.Settings)

	c := &BaseCoder{
		rm:        opts.RepoManager,
		logger:    opts.Logger,
		settings:  opts.Settings,
		history:   history,
		formatter: formatter,
	}

	// Stream processor for the LLMClient wrapper
	var streamProcessor *StreamProcessor
	if opts.Stream {
		streamProcessor = NewStreamProcessor(opts.StreamWriter, opts.Logger)
	}

	// Generate the LLMClient wrapper
	llmClient := NewLLMClient(
		opts.LlmClient,
		opts.Settings,
		opts.Logger,
		streamProcessor,
	)

	c.llmClient = llmClient

	// Should handle this better
	c.SetPrompts(opts.Prompts)
	processor := NewMessageProcessor(
		opts.Logger,
		opts.RepoManager,
		history,
		formatter,
		opts.Settings,
		opts.Prompts,
		c.respExtractor,
	)
	c.processor = processor
	re := make([]string, 0)
	for _, e := range c.respExtractor {
		re = append(re, e.GetName())
	}

	c.logger.Info(
		"setting ",
		"max_output_token", c.settings.GetMaxOutputToken(),
		"model_name", c.settings.GetModelName(),
		"prompt_name",
		opts.Prompts.GetName(),
		"extractors",
		re,
	)
	return c
}

func (c *BaseCoder) SetPrompts(pts prompts.Prompter) {
	c.prompts = pts

	extractors := strings.Split(c.getPrompts().GetEditFormat(), "\n")
	for _, name := range extractors {
		extractor := responseextractor.New(responseextractor.ExtractorType(name), c.logger)
		if extractor == nil {
			c.logger.Error("Error creating extractor", "name", extractor)
			continue
		}
		c.respExtractor = append(c.respExtractor, extractor)
	}

	// c.templateHandler = prompts.NewTemplateHandler(
	// 	c.prompts,
	// 	c.settings,
	// 	c.logger,
	// )
}

func (c *BaseCoder) GetRM() repomanager.RepoManagerInterface {
	return c.rm
}

func (c *BaseCoder) getPrompts() prompts.Prompter {
	return c.prompts
}

func (c *BaseCoder) initBeforeMessage() {
	// Reset state before processing a new message
	// if c.rm.GetGit() != nil {
	// 	// Should commit before message
	// 	lastCommitHash, err := c.rm.GetGit().GetHeadCommitSHA(false)
	// 	if err != nil {
	// 		fmt.Println("Error getting head commit SHA:", err)
	// 	} else {
	// 		c.lastCommitHash = lastCommitHash
	// 	}
	// }
}

func (c *BaseCoder) Run(message string) error {
	c.initBeforeMessage()
	return c.SendMessage(message)
}

func (c *BaseCoder) SendMessage(message string) error {
	// Add user message
	c.history.AddMessage(chat.NewMessage("user", chat.NewTextContent(message)))

	// Format messages with appropriate prompts
	i := 0
	messages := c.FormatMessages()

	for len(c.history.GetCurrentMessages()) > 0 && i < 4 {
		messages.Cur = c.history.GetCurrentMessages()
		tools := make([]chat.Tool, 0)
		for _, e := range c.respExtractor {
			tools = append(tools, e.GetChatTools()...)
		}

		for _, tool := range tools {
			c.logger.Debug("SendMessage: tool", "name", tool.Name)
		}
		for _, m := range messages.AllMessages() {
			c.logger.Debug(
				"SendMessage: msg",
				"role",
				m.Role,
				"is_cacheable",
				m.Content[0].IsCacheable(),
				"content",
				m.Content,
			)
		}
		resp, err := c.llmClient.SendMessages(context.Background(), messages, tools)
		if err != nil {
			return err
		}

		err = c.processor.ProcessResponse(resp)
		if err != nil {
			return fmt.Errorf("error processing response: %w", err)
		}
		i += 1
	}

	return nil
}

// FormatMessages formats all messages for the LLM with appropriate prompts
func (c *BaseCoder) FormatMessages() *ChatChunks {
	chunks := c.formatter.FormatMessages(c.history)

	return chunks
}
