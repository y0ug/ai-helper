package coder

import (
	"fmt"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/coder/actions"
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
	actionQueue       *ActionQueue
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
		actionQueue:       NewActionQueue(),
	}
}

func (mp *MessageProcessor) ProcessResponse(resp *chat.ChatResponse) error {
	if len(resp.Choice) == 0 {
		return fmt.Errorf("no choice returned from LLM")
	}
	choice := resp.Choice[0]
	msg := resp.ToMessageParams()

	if msg.Role != "assistant" {
		return fmt.Errorf("last message should be from assistant")
	}

	mp.history.AddMessage(msg)

	_ = mp.ProcessResponseExtractor(msg)

	if choice.StopReason == "end_turn" {
		mp.history.MoveCurrentToDone("")
	}

	mp.logger.Info(
		"End Processing response",
		"stop",
		choice.StopReason,
		"len(currentMessages)",
		len(mp.history.GetCurrentMessages()),
	)

	for _, msg := range mp.history.GetCurrentMessages() {
		mp.logger.Info("CurrentMessages", "role", msg.Role)
		for _, c := range msg.Content {
			mp.logger.Info("Content", "type", c.Type)
		}
	}

	// We have done all the modification we should handle the commit where?
	return nil
}

func (mp *MessageProcessor) ProcessResponseExtractor(
	msg *chat.ChatMessage,
) []actions.Action[any] {
	allActions := make([]actions.Action[any], 0)

	for _, extractor := range mp.responseExtractor {
		results, err := extractor.Extract(msg)
		if err != nil {
			mp.logger.Error("Error extracting response", "error", err)
		}
		allActions = append(allActions, results...)
	}

	// Enqueue newly extracted actions
	mp.actionQueue.Enqueue(allActions...)

	// Process the queue
	mp.processActionQueue()
	return allActions
}

// This handle should be used to commit after all the changed
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

func (mp *MessageProcessor) processActionQueue() {
	for {
		action, ok := mp.actionQueue.Dequeue()
		if !ok {
			break
		}
		err := mp.handleAction(action)
		if err != nil {
			mp.logger.Error("Failed to handle action", "type", action.Type, "error", err)
			// Decide how to handle errors. Possibly keep going or break.
		}
	}
}

func (mp *MessageProcessor) handleAction(action actions.Action[any]) error {
	mp.logger.Debug("HandleAction invoked", "type", action.Type)

	payload := action.Payload
	switch v := payload.(type) {
	case actions.AwaitUserInput:
		return mp.handleAwaitUserInput(v)
	case actions.ApplyEdit:
		return mp.handleApplyEdit(v)
	case actions.ToolResultAction:
		return mp.handleToolResultAction(v)
	case actions.SendChatMessage:
		return mp.handleSendChatMessage(v)
	case actions.ShellCommand:
		return mp.handleShellCommand(v)
	default:
		mp.logger.Warn("Unknown action type", "actionType", action.Type, "type", fmt.Sprintf("%T", payload))
	}
	return nil
}

func (mp *MessageProcessor) handleShellCommand(shellCommand actions.ShellCommand) error {
	mp.logger.Info("Handling shellCommand", "command", shellCommand.Command)

	actionTools := actiontools.NewActionTools(mp.logger)
	actions, err := actionTools.ShellCommand(shellCommand.Command)
	if err != nil {
		return fmt.Errorf("error running shell command: %w", err)
	}
	mp.actionQueue.Enqueue(actions...)
	return nil
}

func (mp *MessageProcessor) handleSendChatMessage(sendChatMessage actions.SendChatMessage) error {
	mp.logger.Info("Handling sendChatMessage", "message", sendChatMessage.Msg)
	mp.history.AddMessage(&sendChatMessage.Msg)
	return nil
}

func (mp *MessageProcessor) handleToolResultAction(
	toolResultAction actions.ToolResultAction,
) error {
	mp.logger.Info(
		"Handling toolResultAction",
		"toolResultID",
		toolResultAction.ToolResult.ToolUseID,
	)

	// Should we enqueue the next action? or execute it here so we can craft the toolResult message
	// mp.actionQueue.Enqueue(toolResultAction.NextAction)
	toolResult := toolResultAction.ToolResult
	err := mp.handleAction(toolResultAction.NextAction)
	if err != nil {
		toolResult.Content = fmt.Sprintf("Error: %s", err)
	} else {
		toolResult.Content = "Ok."
	}

	mp.history.AddMessage(chat.NewMessage("tool", &toolResult))
	return nil
}

func (mp *MessageProcessor) handleApplyEdit(edit actions.ApplyEdit) error {
	mp.logger.Info("Handling applyEdit", "filename", edit.Filename)

	// Possibly reuse the ActionTools
	actionTools := actiontools.NewActionTools(mp.logger)
	newActions, err := actionTools.ApplyEdits(mp.rm.GetFM(), false, edit)
	if err != nil {
		mp.logger.Error("Error applying edits", "error", err)
		// Could enqueue an error action or handle differently
		return err
	}

	// If `ApplyEdits` returns more actions, enqueue them
	if len(newActions) > 0 {
		mp.actionQueue.Enqueue(newActions...)
	}

	// We should handle the commit when all the files have been updated I think
	// mp.handleSuccessfulEdit()
	return nil
}

func (mp *MessageProcessor) handleAwaitUserInput(userInputAction actions.AwaitUserInput) error {
	mp.logger.Info("Awaiting user input", "question", userInputAction.Question)
	// // 1) Output the question to user
	// mp.history.AddMessage(chat.NewMessage("assistant",
	// 	chat.NewTextContent(userInputAction.Question),
	// ))
	// // 2) Put the conversation in "waiting for user"
	// mp.setAwaitingUserInput(userInputAction.NextAction)
	return nil
}
