package executors

import "log/slog"

type Executor struct {
	logger *slog.Logger
}

func NewExecutor(logger *slog.Logger) *Executor {
	return &Executor{
		logger: logger,
	}
}
