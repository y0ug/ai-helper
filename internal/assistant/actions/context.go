package actions

import (
	"context"
	"log/slog"
)

type contextKey string

const (
	LoggerKey contextKey = "logger"
)

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}

func GetLogger(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(LoggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}
