package coder

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/y0ug/ai-helper/internal/coder/prompts"
	"github.com/y0ug/ai-helper/internal/coder/repomanager"
	"github.com/y0ug/ai-helper/internal/coder/settings"
)

type MessageFormatter struct {
	logger          *slog.Logger
	templateHandler *prompts.TemplateHandler
	rm              repomanager.RepoManagerInterface
	prompts         prompts.Prompter
	settings        *settings.CoderSettings
}

func (mf *MessageFormatter) FormatAllMessages(history *ChatHistory) *ChatChunks {
	mf.rm.ChooseFence()
	mf.settings.Update(mf.rm)
	chunks := &ChatChunks{}

	// Add system messages
	exampleMessages := make([]prompts.Message, 0)

	mainSystem := mf.RenderPrompt(mf.prompts.GetMainSystem())
	if mf.settings.MainModel().ExamplesAsSysMsg {
		if len(mf.prompts.GetExampleMessages()) > 0 {
			mainSystem += "\n# Examples conversations:\n\n"
		}
		for _, msg := range mf.prompts.GetExampleMessages() {
			role := strings.ToUpper(msg.Role)
			content := mf.RenderPrompt(msg.Content)
			mainSystem += fmt.Sprintf("## %s: %s\n\n", role, content)
		}
	} else {
		for _, msg := range mf.prompts.GetExampleMessages() {
			msg.Content = mf.RenderPrompt(msg.Content)
			exampleMessages = append(exampleMessages, msg)
		}

		if len(exampleMessages) > 0 {
			msg := []prompts.Message{{
				Role:    "user",
				Content: "I switched to a new code base. Please don't consider the above files  or try to edit them any longer",
			}, {Role: "assistant", Content: "Ok."}}
			exampleMessages = append(exampleMessages, msg...)
		}
	}
	systemReminder := mf.RenderPrompt(mf.prompts.GetSystemReminder())
	if len(systemReminder) > 0 {
		mainSystem += "\n" + systemReminder
	}

	// msg := chat.NewMessage("system", chat.NewTextContent(mainSystem))
	if mf.settings.MainModel().UseSystemPrompt {
		chunks.System = []prompts.Message{{
			Role:    "system",
			Content: mainSystem,
		}}
	} else {
		chunks.System = []prompts.Message{
			{Role: "user", Content: mainSystem},
			{Role: "assistant", Content: "Ok."},
		}
	}

	chunks.Examples = exampleMessages

	// Summarize end call

	chunks.Done = history.GetDoneMessages()
	chunks.Repo = mf.GetRepoMessages()
	chunks.ReadOnlyFiles = mf.GetReadOnlyFilesMessages()
	chunks.ChatFiles = mf.GetChatFilesMessages()
	chunks.Cur = history.GetCurrentMessages()

	var finalMessage *prompts.Message
	if len(chunks.Cur) > 0 {
		finalMessage = &chunks.Cur[len(chunks.Cur)-1]
	}

	// msgTokens := mf.settings.MainModel().TokenCount(chunks.AllMessages())
	// reminderTokens := mf.settings.MainModel().TokenCount(reminderMessage)
	// curTokens := mf.settings.MainModel().TokenCount(chunks.Cur)
	// totalTokens := msgTokens + reminderTokens + curTokens
	// if totalTokens < maxInputTokens and len(systemReminder)
	// Count if we have enought token to add the reminder
	if len(systemReminder) > 0 {
		if mf.settings.MainModel().Reminder == "sys" {
			chunks.Reminder = []prompts.Message{{
				Role:    "system",
				Content: systemReminder,
			}}
		} else if mf.settings.MainModel().Reminder == "user" && finalMessage != nil && finalMessage.Role == "user" {
			finalMessage.Content = fmt.Sprintf("%s\n\n%s", finalMessage.Content, systemReminder)
		}
	}

	return chunks
}

func NewMessageFormatter(logger *slog.Logger, rm repomanager.RepoManagerInterface,
	p prompts.Prompter, settings *settings.CoderSettings,
) *MessageFormatter {
	return &MessageFormatter{
		logger:          logger,
		rm:              rm,
		prompts:         p,
		settings:        settings,
		templateHandler: prompts.NewTemplateHandler(p, settings, logger),
	}
}

func (mf *MessageFormatter) RenderPrompt(tmpl string) string {
	return mf.templateHandler.Render(tmpl)
}

func (mf *MessageFormatter) RenderPromptData(tmpl string, data map[string]interface{}) string {
	return mf.templateHandler.RenderData(tmpl, data)
}

func (mf *MessageFormatter) GetRepoMessages() []prompts.Message {
	repoContent := mf.rm.GetRepoMap()
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

func (mf *MessageFormatter) GetReadOnlyFilesMessages() []prompts.Message {
	content := mf.rm.GetReadOnlyFilesContent()
	if content == "" {
		return nil
	}

	return []prompts.Message{
		{
			Role:    "user",
			Content: mf.RenderPrompt(mf.prompts.GetReadOnlyFilesPrefix()) + "\n" + content,
		},
		{
			Role:    "assistant",
			Content: "Ok, I will use these files as references.",
		},
	}
}

func (mf *MessageFormatter) GetChatFilesMessages() []prompts.Message {
	if len(mf.rm.GetFM().List(0)) == 0 {
		if mf.rm.GetRepoMap() != "" && mf.prompts.GetFilesNoFullFilesWithRepoMap() != "" {
			return []prompts.Message{
				{
					Role:    "user",
					Content: mf.RenderPrompt(mf.prompts.GetFilesNoFullFilesWithRepoMap()),
				},
				{
					Role:    "assistant",
					Content: mf.RenderPrompt(mf.prompts.GetFilesNoFullFilesWithRepoMapReply()),
				},
			}
		}
		return []prompts.Message{
			{Role: "user", Content: mf.RenderPrompt(mf.prompts.GetFilesNoFullFiles())},
			{Role: "assistant", Content: "Ok."},
		}
	}

	content := mf.RenderPrompt(mf.prompts.GetFilesContentPrefix()) + "\n" + mf.rm.GetFilesContent()

	return []prompts.Message{
		{Role: "user", Content: content},
		{
			Role:    "assistant",
			Content: mf.RenderPrompt(mf.prompts.GetFilesContentAssistantReply()),
		},
	}
}
