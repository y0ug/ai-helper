package assistant

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/actions/executors"
	"github.com/y0ug/ai-helper/internal/assistant/actions/middleware"
	"github.com/y0ug/ai-helper/internal/assistant/actions/queue"
	"github.com/y0ug/ai-helper/internal/assistant/extractors"
	"github.com/y0ug/ai-helper/internal/assistant/prompts"
	"github.com/y0ug/ai-helper/internal/assistant/repomanager"
	"github.com/y0ug/ai-helper/internal/assistant/settings"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type ActionExecutor struct {
	logger        *slog.Logger
	rm            repomanager.RepoManagerInterface
	history       *ChatHistory
	formatter     *PromptFormatter
	settings      *settings.CoderSettings
	extractors    []extractors.Extractor
	prompts       prompts.Prompter
	executor      executors.Executor
	queue         queue.ActionQueuer
	actionManager *actions.ActionManager
}

func NewActionExecutor(logger *slog.Logger, rm repomanager.RepoManagerInterface,
	history *ChatHistory, formatter *PromptFormatter, settings *settings.CoderSettings,
	prompts prompts.Prompter, extractors []extractors.Extractor, executor executors.Executor,
) *ActionExecutor {
	actionManager := actions.NewActionManager(logger)
	return &ActionExecutor{
		logger:        logger,
		rm:            rm,
		history:       history,
		formatter:     formatter,
		settings:      settings,
		prompts:       prompts,
		extractors:    extractors,
		executor:      executor,
		queue:         queue.NewActionQueue(),
		actionManager: actionManager,
	}
}

func (mp *ActionExecutor) ProcessResponse(ctx context.Context, resp *chat.ChatResponse) error {
	if len(resp.Choice) == 0 {
		return fmt.Errorf("no choice returned from LLM")
	}
	choice := resp.Choice[0]
	msg := resp.ToMessageParams()

	if msg.Role != "assistant" {
		return fmt.Errorf("last message should be from assistant")
	}

	mp.history.AddMessage(msg)

	_ = mp.ProcessResponseExtractor(ctx, msg)

	mp.DumpActionChains()

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

func (mp *ActionExecutor) ProcessResponseExtractor(ctx context.Context,
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
	mp.processActionQueue(ctx)
	return allActions
}

func (mp *ActionExecutor) baseHandler(
	ctx context.Context,
	action actions.Action,
) ([]actions.Action, error) {
	handler := mp.executor.GetHandler(action)
	if handler == nil {

		errMsg := fmt.Errorf("no handler for %s action", action.Type)
		mp.logger.Error("No handler", "error", errMsg, "context", action)
		mp.actionManager.AddError(action.Context.ChainID, action, errMsg)

		// Create error followup action
		errorAction := actions.NewLogAction(&action, errMsg.Error())
		mp.actionManager.RegisterAction(errorAction)
		mp.queue.Enqueue(errorAction)

		// we don't return the error we handle it here
		return nil, nil // fmt.Errorf("no handler for action type %s", action.Type)
	}
	return handler.Handle(ctx, action)
}

func (mp *ActionExecutor) processActionQueue(ctx context.Context) {
	chain := middleware.NewMiddlewareChain(
		mp.baseHandler,
		middleware.NewLoggerMiddleware(mp.logger),
	)

	for !mp.queue.IsEmpty() {
		action, _ := mp.queue.Dequeue()

		mp.actionManager.RegisterAction(action)

		// Mark action as completed after processing
		defer func(a *actions.Action) {
			a.Completed = true
		}(&action)
		// Register action with manager before processing
		mp.actionManager.RegisterAction(action)

		results, err := chain.Process(ctx, action)
		if err != nil {
			mp.logger.Error("Error handling action", "error", err, "context", action)
		}

		// Register results and follow-up actions
		for _, result := range results {
			result.Completed = true
			mp.actionManager.RegisterAction(result)
			mp.actionManager.AddResult(action.Context.ChainID,
				fmt.Sprintf("ACTION: %s", action.String())) // Store action summary
		}

		if len(results) > 0 {
			mp.queue.Enqueue(results...)
		}
	}
}

func (mp *ActionExecutor) DumpActionChains() {
	for _, chain := range mp.actionManager.GetAllChains() {
		fmt.Printf("Action Chain: %s\n", chain.ChainID)
		mp.actionManager.DumpActionChainTree(chain.ChainID)

		fmt.Println("Execution Timeline:")
		sortedActions := chain.GetActionsSorted()
		for i, action := range sortedActions {
			status := "✓"
			if containsError(chain.Results, action.ID) {
				status = "✗"
			}
			fmt.Printf("%s [%d] %s\n", status, i+1, action.String())
		}

		fmt.Println("\nDetailed Results:")
		for _, result := range chain.Results {
			fmt.Println("-", result)
		}
		fmt.Println("--------------------")
	}
}

func containsError(results []string, actionID uuid.UUID) bool {
	for _, result := range results {
		if strings.Contains(result, actionID.String()) &&
			(strings.Contains(result, "ERROR") || strings.Contains(result, "failed")) {
			return true
		}
	}
	return false
}
