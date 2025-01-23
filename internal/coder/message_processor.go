package coder

import (
	"fmt"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/coder/prompts"
	"github.com/y0ug/ai-helper/internal/coder/repomanager"
	"github.com/y0ug/ai-helper/internal/coder/settings"
)

type MessageProcessor struct {
	logger    *slog.Logger
	rm        repomanager.RepoManagerInterface
	history   *ChatHistory
	formatter *MessageFormatter
	settings  *settings.CoderSettings
	prompts   prompts.Prompter
}

func NewMessageProcessor(logger *slog.Logger, rm repomanager.RepoManagerInterface,
	history *ChatHistory, formatter *MessageFormatter, settings *settings.CoderSettings,
	prompts prompts.Prompter,
) *MessageProcessor {
	return &MessageProcessor{
		logger:    logger,
		rm:        rm,
		history:   history,
		formatter: formatter,
		settings:  settings,
		prompts:   prompts,
	}
}

func (mp *MessageProcessor) ProcessResponse(messages []chat.ChatMessage) error {
	if len(messages) == 0 {
		return fmt.Errorf("no messages returned from LLM")
	}

	msg := messages[len(messages)-1]
	if msg.Role != "assistant" {
		return fmt.Errorf("last message should be from assistant")
	}

	mp.history.AddMessage(msg)

	isEdit, err := mp.rm.ProcessEdit(msg.Content)
	if err != nil {
		mp.logger.Error("Error processing edit", "error", err)
		return err
	}

	if isEdit {
		if err := mp.handleSuccessfulEdit(); err != nil {
			return err
		}
	}

	return nil
}

func (mp *MessageProcessor) handleSuccessfulEdit() error {
	commitMsg := "apply diff"
	if err := mp.rm.GetFM().Commit(commitMsg); err != nil {
		mp.logger.Error("Error committing", "error", err)
		return err
	}

	data := map[string]interface{}{
		"Hash":    "12345",
		"Message": commitMsg,
	}
	responseMsg := mp.formatter.RenderPromptData(mp.prompts.GetFilesContentGPTEdits(), data)
	mp.history.MoveCurrentToDone(responseMsg)

	return nil
}
