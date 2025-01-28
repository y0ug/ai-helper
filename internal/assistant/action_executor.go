package assistant

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/actions/executors"
	"github.com/y0ug/ai-helper/internal/assistant/actions/middleware"
	"github.com/y0ug/ai-helper/internal/assistant/actions/queue"
	"github.com/y0ug/ai-helper/internal/assistant/eventbus"
	"github.com/y0ug/ai-helper/internal/assistant/extractors"
	"github.com/y0ug/ai-helper/internal/assistant/prompt"
	"github.com/y0ug/ai-helper/internal/assistant/prompt/prompts"
	"github.com/y0ug/ai-helper/internal/assistant/repomanager"
	"github.com/y0ug/ai-helper/internal/assistant/settings"
)

type ActionExecutor struct {
	logger        *slog.Logger
	rm            repomanager.RepoManagerInterface
	history       *prompt.ChatHistory
	formatter     *prompt.PromptFormatter
	settings      *settings.CoderSettings
	extractors    []extractors.Extractor
	prompts       prompts.Prompter
	executor      executors.Executor
	queue         queue.ActionQueuer
	actionManager *actions.ActionManager
	runCtx        context.Context
	runCancel     context.CancelFunc
	eventBus      *eventbus.EventBus
	pipeline      *actions.Pipeline
}

func NewActionExecutor(
	logger *slog.Logger,
	rm repomanager.RepoManagerInterface,
	history *prompt.ChatHistory,
	formatter *prompt.PromptFormatter,
	settings *settings.CoderSettings,
	prompts prompts.Prompter,
	extractors []extractors.Extractor,
	executor executors.Executor,
	bus *eventbus.EventBus,
) *ActionExecutor {
	actionManager := actions.NewActionManager(logger)
	mp := &ActionExecutor{
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
		eventBus:      eventbus.GetEventBus(),
	}
	chain := middleware.NewMiddlewareChain(
		mp.baseHandler,
		middleware.NewLoggerMiddleware(mp.logger),
		middleware.NewTimeoutMiddleware(time.Second*1),
	)
	mp.pipeline = actions.NewPipeline(logger, chain.Process)

	return mp
}

func (mp *ActionExecutor) Start(ctx context.Context) {
	if mp.runCtx == nil {
		mp.runCtx, mp.runCancel = context.WithCancel(ctx)

		// Subscribe to action events
		actionSub := mp.eventBus.Subscribe(100)

		// Catch new EventAction and pass them to EventActionProcess
		go mp.processingLoop(mp.runCtx, actionSub)
		mp.pipeline.Start(mp.runCtx)
	}
}

func (mp *ActionExecutor) Stop() {
	if mp.runCancel != nil {
		mp.runCancel()
	}
	mp.runCancel = nil
	mp.runCtx = nil
}

func (mp *ActionExecutor) processingLoop(ctx context.Context, sub <-chan eventbus.Event) {
	mp.logger.Debug("ActionExecutor: start processing loop")
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-sub:
			if event.Type != eventbus.EventAction {
				continue
			}

			action, ok := event.Payload.(actions.Action)
			if !ok {
				mp.logger.Error("Invalid action payload", "event", event)
				continue
			}

			// mp.processSingleAction(ctx, action)
			mp.logger.Debug("ActionExecutor: Publishing EventActionProcess", "action", action)
			mp.eventBus.Publish(eventbus.NewEvent(eventbus.EventActionProcess, action))
		}
	}
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

// func (mp *ActionExecutor) ProcessResponse(
//
//	ctx context.Context,
//	resp *chat.ChatResponse,
//	parentAction *actions.Action,
//
//	) error {
//		if len(resp.Choice) == 0 {
//			return fmt.Errorf("no choice returned from LLM")
//		}
//		choice := resp.Choice[0]
//		msg := resp.ToMessageParams()
//
//		if msg.Role != "assistant" {
//			return fmt.Errorf("last message should be from assistant")
//		}
//
//		// Create an LLMResponseAction, child of parentAction
//		var rawText string
//		if len(msg.Content) > 0 {
//			rawText = msg.Content[0].String()
//		}
//
//		llmRespAction := actions.NewLLMResponseAction(rawText).WithParent(parentAction)
//		mp.actionManager.RegisterAction(llmRespAction)
//
//		// Add to curernt chat history
//		mp.history.AddMessage(msg)
//
//		// We are pushing to the event action to extract action
//		// from the LLM response. We alsa pass the parent action
//		mp.eventBus.Publish(
//			eventbus.NewEvent(
//				eventbus.EventAction,
//				actions.NewActionExtractor(*msg).WithParent(&llmRespAction),
//			),
//		)
//
//		// If LLM stop reason is end_turn, we move the current messages to done
//		// So any new llm request will start rebuilding the prompt states
//		if choice.StopReason == "end_turn" {
//			mp.history.MoveCurrentToDone("")
//		}
//
//		mp.logger.Info(
//			"End Processing response",
//			"stop",
//			choice.StopReason,
//			"len(currentMessages)",
//			len(mp.history.GetCurrentMessages()),
//		)
//
//		return nil
//	}

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
