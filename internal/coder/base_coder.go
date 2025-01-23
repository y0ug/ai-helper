package coder

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/coder/editservice"
	"github.com/y0ug/ai-helper/internal/coder/models"
	"github.com/y0ug/ai-helper/internal/coder/prompts"
	"github.com/y0ug/ai-helper/internal/coder/repomanager"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
)

type BaseCoder struct {
	mainModel    *models.Model
	editFormat   string
	logger       *slog.Logger
	streamWriter io.Writer
	llmClient    chat.Provider
	// repo       gitrepo.GitRepoInterface
	// fileManager          filemanager.FileManager
	curMessages          []prompts.Message
	doneMessages         []prompts.Message
	maxOutputToken       int
	lastCommitHash       string
	aiderCommitHashes    map[string]struct{}
	temperature          float64
	autoLint             bool
	autoTest             bool
	testCmd              string
	totalCost            float64
	chatLanguage         string
	verbose              bool
	prompts              prompts.Prompter
	lintCommands         map[string]string
	suggestShellCommands bool
	rm                   repomanager.RepoManagerInterface
	templateHandler      *prompts.TemplateHandler
}

func NewBaseCoder(opts CoderOptions) *BaseCoder {
	c := &BaseCoder{
		mainModel:    opts.MainModel,
		llmClient:    opts.LlmClient,
		rm:           opts.RepoManager,
		logger:       opts.Logger,
		curMessages:  make([]prompts.Message, 0),
		doneMessages: make([]prompts.Message, 0),
	}

	c.maxOutputToken = 4096
	if val, ok := c.mainModel.ExtraParams["max_tokens"]; ok {
		switch val := val.(type) {
		case int:
			c.maxOutputToken = val
		}
	}

	c.SetPrompts(opts.Prompts)
	c.logger.Info(
		"setting ",
		"max_output_token", c.maxOutputToken,
		"model_name", c.mainModel.Name,
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
		c.mainModel,
		c.getTemplateInitData(),
		c.logger,
	)
}

func (c *BaseCoder) SetStreamWriter(w io.Writer) {
	c.streamWriter = w
}

func (c *BaseCoder) GetRM() repomanager.RepoManagerInterface {
	return c.rm
}

func (c *BaseCoder) getPrompts() prompts.Prompter {
	return c.prompts
}

func (c *BaseCoder) InitBeforeMessage() {
	// Reset state before processing a new message
	if c.rm.GetGit() != nil {
		// Should commit before message
		lastCommitHash, err := c.rm.GetGit().GetHeadCommitSHA(false)
		if err != nil {
			fmt.Println("Error getting head commit SHA:", err)
		} else {
			c.lastCommitHash = lastCommitHash
		}
	}
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

	// Send to LLM and handle response
	msg, err := c.SendToLLM(messages)
	if err != nil {
		return err
	}

	// if not interrupted
	// Check for file mentions.
	// If files are mentions and we add them start reflected_messages and reprocess
	// else call callback reply_completed

	// Process the response
	for _, m := range msg {
		if m.Role != "assistant" {
			continue
		}

		// Add assistant message to curMessage
		c.curMessages = append(c.curMessages, msg...)

		isEdit, err := c.rm.ProcessEdit(m.Content)
		if err != nil {
			c.logger.Error("Error processing edit", "error", err)
		}
		if isEdit {
			commitMsg := "apply diff"
			err = c.rm.GetFM().Commit(commitMsg)
			if err != nil {
				c.logger.Error("Error committing", "error", err)
			}
			// Should pass commit hash and message
			data := map[string]string{
				"Hash":    "12345",
				"Message": commitMsg,
			}
			responseMsg := c.renderPromptData(c.getPrompts().GetFilesContentGPTEdits(), data)
			c.moveBackCurMessages(responseMsg)
		}

		// Should onlt have one assistant message??
		break
	}

	return nil
	// return c.ApplyEdits(edits)
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

func (c *BaseCoder) processResponse(resp *chat.ChatResponse) ([]prompts.Message, error) {
	// Process response
	msgParams := resp.ToMessageParams()
	// c.logger.Debug("response", "resp", resp)
	c.logger.Debug("msg", "role", msgParams.Role, "content", msgParams.Content)

	// Process response
	return []prompts.Message{{Role: msgParams.Role, Content: msgParams.Content[0].String()}}, nil
}

func (c *BaseCoder) SendToLLM(messages *ChatChunks) ([]prompts.Message, error) {
	messagesLLM := make([]*chat.ChatMessage, 0)
	for m := range messages.System {
		messagesLLM = append(
			messagesLLM,
			chat.NewMessage("system", chat.NewTextContent(messages.System[m].Content)),
		)
	}
	for _, m := range messages.AllMessages() {
		messagesLLM = append(
			messagesLLM,
			chat.NewMessage(m.Role, chat.NewTextContent(m.Content)))
	}
	// Send messages to LLM
	chatParams := chat.NewChatParams(
		chat.WithMaxTokens(c.maxOutputToken),
		chat.WithModel(c.mainModel.Name),
		chat.WithMessages(messagesLLM...))

	ctx := context.Background()
	if c.streamWriter != nil {
		stream, err := c.llmClient.Stream(ctx, *chatParams)
		if err != nil {
			c.logger.Error("Error streaming", "error", err)
			return nil, err
		}

		eventCh := make(chan chat.EventStream)

		// llmclient.ConsumeStreamIO(ctx, stream, os.Stdout)
		go func() {
			// llmclient.ConsumeStreamIO(ctx, stream, os.Stdout)
			if err := chat.StreamChatMessageToChannel(ctx, stream, eventCh); err != nil {
				if err != context.Canceled {
					c.logger.Error("Error consuming stream", "error", err)
				}
			}
		}()

		resp, err := processStream(ctx, c.streamWriter, eventCh)
		if err != nil {
			c.logger.Error("Error processing stream", "error", err)
			return nil, nil
		}

		if resp == nil {
			c.logger.Error("no message return")
			return nil, fmt.Errorf("no message returned from LLM")
		}
		return c.processResponse(resp)
	} else {
		resp, err := c.llmClient.Send(ctx, *chatParams)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			c.logger.Error("no message return")
			return nil, fmt.Errorf("no message returned from LLM")
		}
		return c.processResponse(resp)
	}
	// return nil, nil
}

// FormatMessages formats all messages for the LLM with appropriate prompts
func (c *BaseCoder) FormatMessages() *ChatChunks {
	c.rm.ChooseFence()
	c.templateHandler.SetFence(c.rm.GetFence())
	chunks := &ChatChunks{}

	// Add system messages
	systemPrompt := c.renderPrompt(c.getPrompts().GetMainSystem())
	if c.mainModel.UseSystemPrompt {
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

func (c *BaseCoder) getTemplateInitData() prompts.TemplateData {
	return prompts.TemplateData{
		"Language": c.getLanguage(),
		"Platform": c.getPlatformInfo(),
	}
}

func (c *BaseCoder) renderPrompt(tmpl string) string {
	return c.templateHandler.Render(tmpl)
}

func (c *BaseCoder) renderPromptData(tmpl string, data map[string]string) string {
	return c.templateHandler.RenderData(tmpl, data)
}

func (c *BaseCoder) getLanguage() string {
	if c.chatLanguage != "" {
		return c.chatLanguage
	}
	return "the same language they are using"
}

func processStream(
	ctx context.Context,
	w io.Writer,
	ch <-chan chat.EventStream,
) (*chat.ChatResponse, error) {
	var cm *chat.ChatResponse
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case set, ok := <-ch:
			if !ok {
				return cm, nil
			}
			if set.Type == "text_delta" {
				if w != nil {
					fmt.Fprintf(w, "%v", set.Delta)
				}
			}
			if set.Type == "message_stop" {
				cm = set.Message
			}
		}
	}
}
