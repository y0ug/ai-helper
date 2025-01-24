package executors

import (
	"context"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
)

type LogExecutor struct {
	logger *slog.Logger
}

func NewLogExecutor(
	logger *slog.Logger,
) *LogExecutor {
	return &LogExecutor{
		logger: logger,
	}
}

func (e *LogExecutor) CanHandle(action actions.Action) bool {
	_, ok := action.Payload.(actions.LogAction)
	return ok
}

func (e *LogExecutor) Handle(
	ctx context.Context,
	action actions.Action,
) ([]actions.Action, error) {
	log := action.Payload.(actions.LogAction)
	e.logger.Info("Log", "Message", log.Message)
	return []actions.Action{}, nil
}
