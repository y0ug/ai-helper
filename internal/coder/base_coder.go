package coder

import (
	"context"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/coder/editservice"
	"github.com/y0ug/ai-helper/internal/coder/prompts"
	"github.com/y0ug/ai-helper/internal/coder/repomanager"
	"github.com/y0ug/ai-helper/internal/coder/settings"
)

type BaseCoder struct {
	logger    *slog.Logger
	llmClient *LLMClient
	prompts   prompts.Prompter
	rm        repomanager.RepoManagerInterface
	settings  *settings.CoderSettings
	processor *MessageProcessor
	history   *ChatHistory
	formatter *MessageFormatter
}

func NewBaseCoder(opts CoderOptions) *BaseCoder {
	history := NewChatHistory()
	formatter := NewMessageFormatter(opts.Logger, opts.RepoManager, opts.Prompts, opts.Settings)
	processor := NewMessageProcessor(
		opts.Logger,
		opts.RepoManager,
		history,
		formatter,
		opts.Settings,
		opts.Prompts,
	)

	c := &BaseCoder{
		rm:        opts.RepoManager,
		logger:    opts.Logger,
		settings:  opts.Settings,
		history:   history,
		formatter: formatter,
		processor: processor,
	}

	// Stream processor for the LLMClient wrapper
	streamProcessor := NewStreamProcessor(opts.StreamWriter, opts.Logger)

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

	c.logger.Info(
		"setting ",
		"max_output_token", c.settings.GetMaxOutputToken(),
		"model_name", c.settings.GetModelName(),
		"prompt_name",
		opts.Prompts.GetName(),
		"edit_format",
		opts.RepoManager.GetEditServiceFormat(),
		"edit_svc",
		opts.RepoManager.GetEditServiceName(),
	)
	return c
}

func (c *BaseCoder) SetPrompts(pts prompts.Prompter) {
	c.prompts = pts
	editFormat := editservice.EditFormat(c.getPrompts().GetEditFormat())
	editSvc := editservice.New(editFormat, c.logger)
	if editSvc == nil {
		c.logger.Error("Error creating edit service", "edit_format", editFormat)
	}

	c.rm.SetEditService(editSvc)

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
	c.history.AddMessage(prompts.Message{
		Role:    "user",
		Content: message,
	})

	// Format messages with appropriate prompts
	messages := c.FormatMessages()

	for _, m := range messages.AllMessages() {
		c.logger.Debug("msg", "role", m.Role, "content", m.Content)
	}

	response, err := c.llmClient.SendMessages(context.Background(), messages)
	if err != nil {
		return err
	}

	return c.processor.ProcessResponse(response)
}

// FormatMessages formats all messages for the LLM with appropriate prompts
func (c *BaseCoder) FormatMessages() *ChatChunks {
	chunks := c.formatter.FormatAllMessages()

	// Add chat history
	chunks.Done = c.history.GetDoneMessages()

	// Add repo content if available
	if repoMsgs := c.formatter.GetRepoMessages(); len(repoMsgs) > 0 {
		chunks.Repo = repoMsgs
	}

	// Add readonly files content
	if readOnlyMsgs := c.formatter.GetReadOnlyFilesMessages(); len(readOnlyMsgs) > 0 {
		chunks.ReadOnlyFiles = readOnlyMsgs
	}

	// Add chat files content
	if chatFilesMsgs := c.formatter.GetChatFilesMessages(); len(chatFilesMsgs) > 0 {
		chunks.ChatFiles = chatFilesMsgs
	}

	// Add current conversation
	chunks.Cur = c.history.GetCurrentMessages()

	return chunks
}
