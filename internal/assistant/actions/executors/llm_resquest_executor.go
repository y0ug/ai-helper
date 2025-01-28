package executors

import (
	"context"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
)

type (
	Processor[T any]   func(context.Context, T) ([]T, error)
	LLMRequestExecutor struct {
		cb Processor[actions.Action]
	}
)

func NewLLMRequestExecutor(
	cb Processor[actions.Action],
) *LLMRequestExecutor {
	return &LLMRequestExecutor{
		cb: cb,
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
	// logger := actions.GetLogger(ctx)

	_ = action.Payload.(actions.LLMRequestAction)
	// resp := val.Resp

	return e.cb(ctx, action)
}
