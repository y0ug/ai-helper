package executors

import (
	"context"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/prompt"
)

type AddMessageExecutor struct {
	history *prompt.ChatHistory
}

func NewAddMessageExecutor(history *prompt.ChatHistory) *AddMessageExecutor {
	return &AddMessageExecutor{
		history: history,
	}
}

func (e *AddMessageExecutor) CanHandle(action actions.Action) bool {
	_, ok := action.Payload.(actions.AddMessageAction)
	return ok
}

func (e *AddMessageExecutor) Handle(
	ctx context.Context,
	action actions.Action,
) ([]actions.Action, error) {
	logger := actions.GetLogger(ctx)

	val := action.Payload.(actions.AddMessageAction)
	logger.Info("AddMessageExecutor", "Role", val.Msg.Role, "Content", val.Msg.Content)

	e.history.AddMessage(&val.Msg)
	return []actions.Action{actions.NewLLMRequestAction("foo").WithParent(&action)}, nil
}
