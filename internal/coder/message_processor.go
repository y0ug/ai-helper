package coder

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/coder/actiontools"
	"github.com/y0ug/ai-helper/internal/coder/prompts"
	"github.com/y0ug/ai-helper/internal/coder/repomanager"
	"github.com/y0ug/ai-helper/internal/coder/responseextractor"
	"github.com/y0ug/ai-helper/internal/coder/settings"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type MessageProcessor struct {
	logger            *slog.Logger
	rm                repomanager.RepoManagerInterface
	history           *ChatHistory
	formatter         *MessageFormatter
	settings          *settings.CoderSettings
	responseExtractor []responseextractor.ResponseExtractor
	prompts           prompts.Prompter
}

func NewMessageProcessor(logger *slog.Logger, rm repomanager.RepoManagerInterface,
	history *ChatHistory, formatter *MessageFormatter, settings *settings.CoderSettings,
	prompts prompts.Prompter, responseExtractor []responseextractor.ResponseExtractor,
) *MessageProcessor {
	return &MessageProcessor{
		logger:            logger,
		rm:                rm,
		history:           history,
		formatter:         formatter,
		settings:          settings,
		prompts:           prompts,
		responseExtractor: responseExtractor,
	}
}

func (mp *MessageProcessor) ProcessResponse(resp *chat.ChatResponse) error {
	// Should we handle the response of the message only?
	// Where should we extract the usage that is in the chat.ChatResponse USage
	if len(resp.Choice) == 0 {
		return fmt.Errorf("no choice returned from LLM")
	}
	choice := resp.Choice[0]
	msg := resp.ToMessageParams()

	if msg.Role != "assistant" {
		return fmt.Errorf("last message should be from assistant")
	}

	mp.history.AddMessage(msg)

	actions := mp.ProcessResponseExtractor(msg)
	mp.logger.Info("Processing response", "stop", choice.StopReason, "len(actions)", len(actions))
	// for _, action := range actions {
	// 	mp.logger.Debug("action", "type", action.Type, "json", string(action.Input))
	// }
	mp.history.MoveCurrentToDone("")

	// state, contents, err := mp.rm.Process(msg)
	// mp.logger.Info("Processing response", "stop", choice.StopReason, "len(content)", len(contents))
	// if err != nil {
	// 	mp.logger.Error("Error processing edit", "error", err)
	// 	return err
	// }
	// if len(contents) > 0 {
	//
	// 	c := make([]*chat.MessageContent, 0)
	// 	for _, content := range contents {
	// 		c = append(c, &content)
	// 	}
	//
	// 	respMsg := chat.NewMessage("tool", c...)
	// 	mp.handleToolResult(respMsg)
	// } else if state == repomanager.ProcessTypeEdit {
	// 	if err := mp.handleSuccessfulEdit(); err != nil {
	// 		return err
	// 	}
	// } else {
	// 	mp.history.MoveCurrentToDone("")
	// 	mp.logger.Info("MoveCurrentToDone")
	// }

	return nil
}

func (mp *MessageProcessor) ProcessAction(
	actions ...responseextractor.ParsedAction,
) ([]responseextractor.ParsedAction, error) {
	actionTools := actiontools.NewActionTools(mp.logger)

	for _, action := range actions {
		mp.logger.Debug("action", "type", action.Type, "json", string(action.Input))
		if action.Type == responseextractor.ActionApplyEdit {
			var data responseextractor.Edit
			json.Unmarshal(action.Input, &data)
			newActions, err := actionTools.ApplyEdits(mp.rm.GetFM(), true, data)
			if err != nil {
				mp.logger.Error("Error applying edits", "error", err)
				continue
			}
			_, err = mp.ProcessAction(newActions...)
			if err != nil {
				mp.logger.Error("Error applying new action", "error", err)
				continue
			}
		}

		if action.Type == responseextractor.ActionApplyEditToolResult {
			var toolResult chat.MessageContent
			json.Unmarshal(action.Input, &toolResult)

			var a responseextractor.ParsedAction
			json.Unmarshal([]byte(toolResult.Content), &a)

			_, err := mp.ProcessAction(a)
			if err != nil {
				mp.logger.Error("Error applying new action", "error", err)
				continue
			}

			// Send tools result
		}

	}

	return nil, nil
}

func (mp *MessageProcessor) ProcessResponseExtractor(
	msg *chat.ChatMessage,
) []responseextractor.ParsedAction {
	r := make([]responseextractor.ParsedAction, 0)
	for _, re := range mp.responseExtractor {
		results, err := re.Extract(msg)
		if err != nil {
			mp.logger.Error("Error extracting response", "error", err)
		}
		r = append(r, results...)
	}
	mp.ProcessAction(r...)
	return r
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

func (mp *MessageProcessor) handleToolResult(toolResultMsg *chat.ChatMessage) error {
	mp.history.AddMessage(toolResultMsg)
	return nil
}
