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
	curMessages     []prompts.Message
	doneMessages    []prompts.Message
	prompts         prompts.Prompter
	rm              repomanager.RepoManagerInterface
	templateHandler *prompts.TemplateHandler
	settings        *settings.CoderSettings
}

func NewBaseCoder(opts CoderOptions) *BaseCoder {
	c := &BaseCoder{
		// llmClient:    opts.LlmClient,
		rm:           opts.RepoManager,
		logger:       opts.Logger,
		curMessages:  make([]prompts.Message, 0),
		doneMessages: make([]prompts.Message, 0),
		settings:     opts.Settings,
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
	c.curMessages = append(c.curMessages, prompts.Message{
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

	return c.handleResponse(response)
}

func (c *BaseCoder) handleResponse(messages []prompts.Message) error {
	if len(messages) == 0 {
		return fmt.Errorf("no messages returned from LLM")
	}

	msg := messages[len(messages)-1]
	if msg.Role != "assistant" {
		return fmt.Errorf("last message should be from assistant")
	}
	// Add assistant message to curMessage
	c.curMessages = append(c.curMessages, msg)

	isEdit, err := c.rm.ProcessEdit(msg.Content)
	if err != nil {
		c.logger.Error("Error processing edit", "error", err)
		return err
	}

	if isEdit {
		if err := c.handleSuccessfulEdit(); err != nil {
			return err
		}
	}

	return nil
}

func (c *BaseCoder) handleSuccessfulEdit() error {
	commitMsg := "apply diff"
	if err := c.rm.GetFM().Commit(commitMsg); err != nil {
		c.logger.Error("Error committing", "error", err)
		return err
	}

	// Should pass commit hash and message
	data := map[string]interface{}{
		"Hash":    "12345",
		"Message": commitMsg,
	}
	responseMsg := c.renderPromptData(c.getPrompts().GetFilesContentGPTEdits(), data)
	c.moveBackCurMessages(responseMsg)

	return nil
}

func (c *BaseCoder) moveBackCurMessages(message string) {
	c.logger.Debug("adding", "message", message, "curMessage", c.curMessages)
	// Clear current messages if everyting was done
	c.doneMessages = append(c.doneMessages, c.curMessages...)
	c.curMessages = make([]prompts.Message, 0)

	if message != "" {
		c.doneMessages = append(c.doneMessages, prompts.Message{
			Role:    "user",
			Content: message,
		}, prompts.Message{
			Role:    "assistant",
			Content: "Ok.",
		})
	}
}

// FormatMessages formats all messages for the LLM with appropriate prompts
func (c *BaseCoder) FormatMessages() *ChatChunks {
	c.GetRM().ChooseFence()
	c.settings.Update(c.GetRM())
	chunks := &ChatChunks{}

	// Add system messages
	systemPrompt := c.renderPrompt(c.getPrompts().GetMainSystem())
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
		msg.Content = c.renderPrompt(msg.Content)
		chunks.Examples = append(chunks.Examples, msg)
	}

	// Add chat history
	chunks.Done = c.doneMessages

	// Add repo content if available
	if repoMsgs := c.getRepoMessages(); len(repoMsgs) > 0 {
		chunks.Repo = repoMsgs
	}

	// Add readonly files content
	if readOnlyMsgs := c.getReadOnlyFilesMessages(); len(readOnlyMsgs) > 0 {
		chunks.ReadOnlyFiles = readOnlyMsgs
	}

	// Add chat files content
	if chatFilesMsgs := c.getChatFilesMessages(); len(chatFilesMsgs) > 0 {
		chunks.ChatFiles = chatFilesMsgs
	}

	// Add current conversation
	chunks.Cur = c.curMessages

	// Add reminder if needed
	if reminder := c.getPrompts().GetSystemReminder(); reminder != "" {
		chunks.Reminder = []prompts.Message{{
			Role:    "system",
			Content: c.renderPrompt(reminder),
		}}
	}

	return chunks
}

func (c *BaseCoder) getRepoMessages() []prompts.Message {
	repoContent := c.rm.GetRepoMap()
	if repoContent == "" {
		return nil
	}

	return []prompts.Message{
		{
			Role:    "user",
			Content: repoContent,
		},
		{
			Role:    "assistant",
			Content: "Ok, I won't try and edit those files without asking first.",
		},
	}
}

func (c *BaseCoder) getReadOnlyFilesMessages() []prompts.Message {
	content := c.rm.GetReadOnlyFilesContent()
	if content == "" {
		return nil
	}

	return []prompts.Message{
		{
			Role:    "user",
			Content: c.renderPrompt(c.getPrompts().GetReadOnlyFilesPrefix()) + "\n" + content,
		},
		{
			Role:    "assistant",
			Content: "Ok, I will use these files as references.",
		},
	}
}

func (c *BaseCoder) getChatFilesMessages() []prompts.Message {
	if len(c.rm.GetFM().List(0)) == 0 {
		if c.rm.GetRepoMap() != "" && c.getPrompts().GetFilesNoFullFilesWithRepoMap() != "" {
			return []prompts.Message{
				{
					Role:    "user",
					Content: c.renderPrompt(c.getPrompts().GetFilesNoFullFilesWithRepoMap()),
				},
				{
					Role:    "assistant",
					Content: c.renderPrompt(c.getPrompts().GetFilesNoFullFilesWithRepoMapReply()),
				},
			}
		}
		return []prompts.Message{
			{Role: "user", Content: c.renderPrompt(c.getPrompts().GetFilesNoFullFiles())},
			{Role: "assistant", Content: "Ok."},
		}
	}

	content := c.renderPrompt(
		c.getPrompts().GetFilesContentPrefix(),
	) + "\n" + c.rm.GetFilesContent()

	return []prompts.Message{
		{Role: "user", Content: content},
		{
			Role:    "assistant",
			Content: c.renderPrompt(c.getPrompts().GetFilesContentAssistantReply()),
		},
	}
}

func (c *BaseCoder) renderPrompt(tmpl string) string {
	return c.templateHandler.Render(tmpl)
}

func (c *BaseCoder) renderPromptData(tmpl string, data map[string]interface{}) string {
	return c.templateHandler.RenderData(tmpl, data)
}
