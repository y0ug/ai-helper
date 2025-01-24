package executors

import (
	"log/slog"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/filemanager"
)

type Executor interface {
	ApplyEditsHandler(
		fm filemanager.FileManager,
		dryrun bool,
		edits ...actions.ApplyEdit,
	) ([]actions.Action[any], error)
	ShellCommandHandler(
		command string,
	) ([]actions.Action[any], error)
}
type ExecutorLocal struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *ExecutorLocal {
	return &ExecutorLocal{
		logger: logger,
	}
}
