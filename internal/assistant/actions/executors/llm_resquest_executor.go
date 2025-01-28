package executors

import (
	"context"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/eventbus"
	"github.com/y0ug/ai-helper/internal/assistant/prompt"
)

type LLMRequestExecutor struct {
	history *prompt.ChatHistory
}

func NewLLMRequestExecutor(history *prompt.ChatHistory) *LLMRequestExecutor {
	return &LLMRequestExecutor{
		history: history,
	}
}

func (e *LLMRequestExecutor) CanHandle(action actions.Action) bool {
	_, ok := action.Payload.(actions.LLMRequestAction)
	return ok
}

func (e *LLMRequestExecutor) Handle(
	ctx context.Context,
	action actions.Action,
) (results []actions.Action, err error) {
	logger := actions.GetLogger(ctx)

	_ = action.Payload.(actions.LLMRequestAction)
	// resp := val.Resp

	eventbus.GetEventBus().Publish(eventbus.NewEvent(eventbus.EventLLMRequest, action))

	logger.Info(
		"Processing LLM request", "curMessage",
		len(e.history.GetCurrentMessages()),
	)
	return results, nil
}
