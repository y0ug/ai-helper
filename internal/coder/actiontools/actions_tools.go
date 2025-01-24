package actiontools

import "log/slog"

type ActionTools struct {
	logger *slog.Logger
}

func NewActionTools(logger *slog.Logger) *ActionTools {
	return &ActionTools{
		logger: logger,
	}
}
