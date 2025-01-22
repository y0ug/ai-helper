package coder

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/coder/editservice"
	"github.com/y0ug/ai-helper/internal/coder/models"
	"github.com/y0ug/ai-helper/internal/coder/prompts"
	"github.com/y0ug/ai-helper/internal/coder/repomanager"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
)

type BaseCoder struct {
	mainModel  *models.Model
	editFormat string
	logger     *slog.Logger
	llmClient  chat.Provider
	// repo       gitrepo.GitRepoInterface
	// fileManager          filemanager.FileManager
	curMessages          []prompts.Message
	doneMessages         []prompts.Message
	lastCommitHash       string
	aiderCommitHashes    map[string]struct{}
	temperature          float64
	autoLint             bool
	autoTest             bool
	testCmd              string
	totalCost            float64
	chatLanguage         string
	verbose              bool
	prompts              prompts.BasePrompts
	lintCommands         map[string]string
	suggestShellCommands bool
	rm                   repomanager.RepoManagerInterface
}

func NewBaseCoder(opts CoderOptions) *BaseCoder {
	c := &BaseCoder{
		mainModel:    opts.MainModel,
		llmClient:    opts.LlmClient,
		rm:           opts.RepoManager,
		prompts:      *prompts.NewBasePrompts(),
		curMessages:  make([]prompts.Message, 0),
		doneMessages: make([]prompts.Message, 0),
	}
	return c
}

func (c *BaseCoder) getPrompts() *prompts.BasePrompts {
	return &c.prompts
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

	// Send to LLM and handle response
	response, err := c.SendToLLM(messages)
	if err != nil {
		return err
	}

	c.logger.Info("response", "role", response[0].Role, "content", response[0].Content)
	// Handle function call
	// Show usage
	// Check file mention

	// Process response and apply edits
	// edits, err := c.GetEdits(response)
	// if err != nil {
	// 	return err
	// }

	return nil
	// return c.ApplyEdits(edits)
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
		chat.WithMaxTokens(c.mainModel.MaxChatHistoryTokens),
		chat.WithModel(c.mainModel.Name),
		chat.WithMessages(messagesLLM...))

	ctx := context.Background()
	response, err := c.llmClient.Send(ctx, *chatParams)
	if err != nil {
		return nil, err
	}

	msgParams := response.ToMessageParams()
	m := prompts.Message{
		Role:    msgParams.Role,
		Content: msgParams.Content[0].String(),
	}
	c.logger.Info("msg", "role", m.Role, "content", m.Content)
	// Process response
	return []prompts.Message{m}, nil
}

// FormatMessages formats all messages for the LLM with appropriate prompts
func (c *BaseCoder) FormatMessages() *ChatChunks {
	c.rm.ChooseFence()
	chunks := &ChatChunks{}

	// Add system messages
	systemPrompt := c.formatSystemPrompt()
	hasSystemPrompt := true
	if hasSystemPrompt {
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
	chunks.Examples = c.getPrompts().ExampleMessages

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
	if reminder := c.getPrompts().SystemReminder; reminder != "" {
		chunks.Reminder = []prompts.Message{{
			Role:    "system",
			Content: reminder,
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

	promptsR := c.getPrompts()
	return []prompts.Message{
		{
			Role:    "user",
			Content: promptsR.ReadOnlyFilesPrefix + "\n" + content,
		},
		{
			Role:    "assistant",
			Content: "Ok, I will use these files as references.",
		},
	}
}

func (c *BaseCoder) getChatFilesMessages() []prompts.Message {
	if len(c.rm.GetFM().List(0)) == 0 {
		promptsR := c.getPrompts()
		if c.rm.GetRepoMap() != "" && promptsR.FilesNoFullFilesWithRepoMap != "" {
			return []prompts.Message{
				{Role: "user", Content: promptsR.FilesNoFullFilesWithRepoMap},
				{Role: "assistant", Content: promptsR.FilesNoFullFilesWithRepoMapReply},
			}
		}
		return []prompts.Message{
			{Role: "user", Content: promptsR.FilesNoFullFiles},
			{Role: "assistant", Content: "Ok."},
		}
	}

	promptsR := c.getPrompts()
	content := promptsR.FilesContentPrefix + "\n" + c.rm.GetFilesContent()

	return []prompts.Message{
		{Role: "user", Content: content},
		{Role: "assistant", Content: promptsR.FilesContentAssistantReply},
	}
}

func (c *BaseCoder) formatSystemPrompt() string {
	promptsR := c.getPrompts()
	data := TemplateData{
		Language:       c.getLanguage(),
		LazyPrompt:     c.getLazyPrompt(),
		Platform:       c.getPlatformInfo(),
		ShellCmdPrompt: c.getShellCmdPrompt(),
		Fence:          c.rm.GetFence(),
	}

	formatted, err := prompts.RenderTemplate(promptsR.MainSystem, data)
	if err != nil {
		return promptsR.MainSystem
	}
	return formatted
}

func (c *BaseCoder) getLanguage() string {
	if c.chatLanguage != "" {
		return c.chatLanguage
	}
	return "the same language they are using"
}

func (c *BaseCoder) getLazyPrompt() string {
	if c.mainModel.Lazy {
		return c.getPrompts().LazyPrompt
	}
	return ""
}

type TemplateData struct {
	Language       string
	LazyPrompt     string
	Platform       string
	ShellCmdPrompt string
	Fence          editservice.Fence
}
