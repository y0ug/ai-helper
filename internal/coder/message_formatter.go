package coder

import (
	"log/slog"

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
