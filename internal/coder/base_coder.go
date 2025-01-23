package coder

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/coder/editservice"
	"github.com/y0ug/ai-helper/internal/coder/prompts"
	"github.com/y0ug/ai-helper/internal/coder/repomanager"
	"github.com/y0ug/ai-helper/internal/coder/settings"
)

type BaseCoder struct {
	logger          *slog.Logger
	llmClient       *LLMClient
	prompts         prompts.Prompter
	rm              repomanager.RepoManagerInterface
	settings        *settings.CoderSettings
	processor       *MessageProcessor
	history         *ChatHistory
	formatter       *MessageFormatter
}

func NewBaseCoder(opts CoderOptions) *BaseCoder {
	history := NewChatHistory()
	formatter := NewMessageFormatter(opts.Logger, opts.RepoManager, opts.Prompts, opts.Settings)
	
	c := &BaseCoder{
		rm:       opts.RepoManager,
		logger:   opts.Logger,
		settings: opts.Settings,
		history:  history,
		formatter: formatter,
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

	c.SetPrompts(opts.Prompts)
	c.logger.Info(
		"setting ",
		"max_output_token", c.settings.GetMaxOutputToken(),
		"model_name", c.settings.GetModelName(),
		"prompt_name",
		c.getPrompts().GetName(),
		"edit_format",
		c.GetRM().GetEditServiceFormat(),
		"edit_svc",
		c.GetRM().GetEditServiceName(),
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

	c.templateHandler = prompts.NewTemplateHandler(
		c.prompts,
		c.settings,
		c.logger,
	)
}

func (c *BaseCoder) GetRM() repomanager.RepoManagerInterface {
	return c.rm
}

func (c *BaseCoder) getPrompts() prompts.Prompter {
	return c.prompts
}

func (c *BaseCoder) InitBeforeMessage() {
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
	c.InitBeforeMessage()
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
	c.GetRM().ChooseFence()
	c.settings.Update(c.GetRM())
	chunks := &ChatChunks{}

	// Add system messages
	systemPrompt := c.formatter.RenderPrompt(c.getPrompts().GetMainSystem())
	if c.settings.MainModel().UseSystemPrompt {
		chunks.System = []prompts.Message{{
			Role:    "system",
			Content: systemPrompt,
		}}
	} else {
		chunks.System = []prompts.Message{
			{Role: "user", Content: systemPrompt},
			{Role: "assistant", Content: "Ok."},
		}
	}

	// Add example messages from prompts
	for _, msg := range c.getPrompts().GetExampleMessages() {
		msg.Content = c.formatter.RenderPrompt(msg.Content)
		chunks.Examples = append(chunks.Examples, msg)
	}

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

	// Add reminder if needed
	if reminder := c.getPrompts().GetSystemReminder(); reminder != "" {
		chunks.Reminder = []prompts.Message{{
			Role:    "system",
			Content: c.formatter.RenderPrompt(reminder),
		}}
	}

	return chunks
}
