package assistant

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/executors"
	"github.com/y0ug/ai-helper/internal/assistant/extractors"
	"github.com/y0ug/ai-helper/internal/assistant/prompts"
	"github.com/y0ug/ai-helper/internal/assistant/repomanager"
	"github.com/y0ug/ai-helper/internal/assistant/settings"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type ActionExecutor struct {
	logger     *slog.Logger
	rm         repomanager.RepoManagerInterface
	history    *ChatHistory
	formatter  *PromptFormatter
	settings   *settings.CoderSettings
	extractors []extractors.Extractor
	prompts    prompts.Prompter
	executor   executors.Executor
	queue      ActionQueuer
}

func NewActionExecutor(logger *slog.Logger, rm repomanager.RepoManagerInterface,
	history *ChatHistory, formatter *PromptFormatter, settings *settings.CoderSettings,
	prompts prompts.Prompter, extractors []extractors.Extractor, executor executors.Executor,
) *ActionExecutor {
	return &ActionExecutor{
		logger:     logger,
		rm:         rm,
		history:    history,
		formatter:  formatter,
		settings:   settings,
		prompts:    prompts,
		extractors: extractors,
		executor:   executor,
		queue:      NewActionQueue(),
	}
}

func (mp *ActionExecutor) ProcessResponse(resp *chat.ChatResponse) error {
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

func (mp *ActionExecutor) ProcessResponseExtractor(
	msg *chat.ChatMessage,
) []actions.Action[any] {
	allActions := make([]actions.Action[any], 0)

	for _, extractor := range mp.extractors {
		results, err := extractor.Extract(msg)
		if err != nil {
			mp.logger.Error("Error extracting response", "error", err)
		}
		allActions = append(allActions, results...)
	}

	// Enqueue newly extracted actions
	mp.queue.Enqueue(allActions...)

	// Process the queue
	mp.processActionQueue()
	return allActions
}

// This handle should be used to commit after all the changed
func (mp *ActionExecutor) handleSuccessfulEdit() error {
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

func (mp *ActionExecutor) handleToolResult(toolResultMsg *chat.ChatMessage) error {
	mp.history.AddMessage(toolResultMsg)
	return nil
}

func (mp *ActionExecutor) processActionQueue() {
	for {
		action, ok := mp.queue.Dequeue()
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

func (mp *ActionExecutor) handleAction(action actions.Action[any]) error {
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

func (mp *ActionExecutor) handleShellCommand(shellCommand actions.ShellCommand) error {
	mp.logger.Info("Handling shellCommand", "command", shellCommand.Command)

	newActions, err := mp.executor.ShellCommandHandler(shellCommand.Command)
	if err != nil {
		return fmt.Errorf("error running shell command: %w", err)
	}
	mp.queue.Enqueue(newActions...)
	return nil
}

func (mp *ActionExecutor) handleSendChatMessage(sendChatMessage actions.SendChatMessage) error {
	mp.logger.Info("Handling sendChatMessage", "message", sendChatMessage.Msg)
	mp.history.AddMessage(&sendChatMessage.Msg)
	return nil
}

func (mp *ActionExecutor) handleToolResultAction(
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

	toolResultContent := "Ok."
	for _, action := range toolResultAction.NextAction {
		err := mp.handleAction(action)
		if err != nil {
			toolResultContent = fmt.Sprintf("Error: %s", err)
		}
	}

	toolResult.Content = toolResultContent

	mp.history.AddMessage(chat.NewMessage("tool", &toolResult))
	return nil
}

func (mp *ActionExecutor) handleApplyEdit(edit actions.ApplyEdit) error {
	mp.logger.Info("Handling applyEdit", "filename", edit.Filename)

	newActions, err := mp.executor.ApplyEditsHandler(mp.rm.GetFM(), false, edit)
	if err != nil {
		mp.logger.Error("Error applying edits", "error", err)
		// Could enqueue an error action or handle differently
		return err
	}

	// push AwaitUserInput action
	newAction := actions.NewParsedAction(actions.AwaitUserInput{
		Question:  "Do you want to commit the changes?",
		InputType: actions.UserInputTypeConfirm,
	})
	newActions = append(newActions, newAction)

	// If `ApplyEdits` returns more actions, enqueue them
	if len(newActions) > 0 {
		mp.queue.Enqueue(newActions...)
	}

	// We should handle the commit when all the files have been updated I think
	// mp.handleSuccessfulEdit()
	return nil
}

func (mp *ActionExecutor) handleAwaitUserInput(userInputAction actions.AwaitUserInput) error {
	mp.logger.Info(
		"Awaiting user input",
		"question",
		userInputAction.Question,
		"type",
		userInputAction.InputType,
	)

	fmt.Println("QUESTION:", userInputAction.Question)
	var userAnswer string
	if _, err := fmt.Scanln(&userAnswer); err != nil {
		mp.logger.Error("Failed reading input", "error", err)
		return err
	}

	switch userInputAction.InputType {
	case actions.UserInputTypeText:
		mp.logger.Info("We should process the response")
		// 1) Output the question to user
		// mp.history.AddMessage(chat.NewMessage("assistant",
		// 	chat.NewTextContent(userInputAction.Question),
		// ))
	case actions.UserInputTypeConfirm:
		if strings.EqualFold(userAnswer, "yes") {
			mp.queue.Enqueue(userInputAction.NextAction...)
		} else {
			mp.logger.Info("User did not confirm")
		}
	}

	return nil
}
