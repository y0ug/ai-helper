package coder

import (
	"fmt"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/coder/prompts"
	"github.com/y0ug/ai-helper/internal/coder/repomanager"
	"github.com/y0ug/ai-helper/internal/coder/settings"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
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

func (mp *MessageProcessor) ProcessResponse(messages []*chat.ChatMessage) error {
	// Should we handle the response of the message only?
	// Where should we extract the usage that is in the chat.ChatResponse USage
	if len(messages) == 0 {
		return fmt.Errorf("no messages returned from LLM")
	}

	msg := messages[len(messages)-1]
	if msg.Role != "assistant" {
		return fmt.Errorf("last message should be from assistant")
	}

	mp.history.AddMessage(msg)

	state, contents, err := mp.rm.Process(msg)
	if err != nil {
		mp.logger.Error("Error processing edit", "error", err)
		return err
	}

	if len(contents) > 0 {
		c := make([]*chat.MessageContent, 0)
		for _, content := range contents {
			c = append(c, &content)
		}

		respMsg := chat.NewMessage("tool", c...)
		mp.history.AddMessage(respMsg)

	}
	if state == repomanager.ProcessTypeEdit {
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
