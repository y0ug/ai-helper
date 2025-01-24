package assistant

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/actions/executors"
	"github.com/y0ug/ai-helper/internal/assistant/actions/queue"
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
	queue      queue.ActionQueuer
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
		queue:      queue.NewActionQueue(),
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
) []actions.Action {
	allActions := make([]actions.Action, 0)

	for _, extractor := range mp.extractors {
		results, err := extractor.Extract(msg)
		mp.logger.Info("Results", "results", results, "namne", extractor.Name())

		if err != nil {
			mp.logger.Error("Error extracting response", "error", err)
		}
		allActions = append(allActions, results...)
	}

	// Enqueue newly extracted actions
	mp.queue.Enqueue(allActions...)

	mp.logger.Info("queue", "isEmpty", mp.queue.IsEmpty(), "Length", mp.queue.Len())
	// Process the queue
	mp.processActionQueue()
	return allActions
}

func (mp *ActionExecutor) processActionQueue() {
	ctx := context.TODO()
	for !mp.queue.IsEmpty() {
		action, _ := mp.queue.Dequeue()
		handler := mp.executor.GetHandler(action)
		if handler == nil {
			mp.logger.Error("No handler found for action type", "type", action.Type)
			continue
		}
		results, err := handler.Handle(ctx, action)
		if err != nil {
			mp.logger.Error("Error handling action", "error", err)
		}
		// for _, result := range results {
		// 	mp.logger.Debug("Result Handle", "result", result)
		// }

		// Enqueue any resulting actions
		if len(results) > 0 {
			mp.logger.Debug("Enqueuing follow-up actions", "count", len(results))
			mp.queue.Enqueue(results...)
		}

		// // Track results
		// if action.Context.ToolCallID != "" {
		// 	for _, result := range results {
		// 		toolCallManager.AddResult(action.Context.ToolCallID, result)
		// 	}
		// }
	}
}
