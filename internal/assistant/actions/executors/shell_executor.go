package executors

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"time"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
)

type ShellExecutor struct {
	safePatterns   []*regexp.Regexp
	confirmChan    chan<- actions.Action
	logger         *slog.Logger
	commandTimeout time.Duration
}

func NewShellExecutor(
	safePatterns []string,
	confirmChan chan<- actions.Action,
	logger *slog.Logger,
) *ShellExecutor {
	compiled := make([]*regexp.Regexp, len(safePatterns))
	for i, p := range safePatterns {
		compiled[i] = regexp.MustCompile(p)
	}

	return &ShellExecutor{
		safePatterns:   compiled,
		confirmChan:    confirmChan,
		logger:         logger,
		commandTimeout: 2 * time.Minute,
	}
}

func (s *ShellExecutor) CanHandle(action actions.Action) bool {
	_, ok := action.Payload.(actions.ShellCommandAction)
	return ok
}

func (s *ShellExecutor) Handle(
	ctx context.Context,
	action actions.Action,
) ([]actions.Action, error) {
	cmd := action.Payload.(actions.ShellCommandAction)

	// if cmd.NeedsConfirm && !cmd.Confirmed {
	// 	s.confirmChan <- actions.NewUserConfirmAction(&action,
	// 		fmt.Sprintf("Allow command: %s?", cmd.Command))
	// 	return nil, nil
	// }

	if !s.isCommandAllowed(cmd.Command) {
		return []actions.Action{
			actions.NewLogAction(&action, fmt.Sprintf("Blocked unsafe command: %s", cmd.Command)),
		}, fmt.Errorf("unsafe command")
	}

	ctx, cancel := context.WithTimeout(ctx, s.commandTimeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, "sh", "-c", cmd.Command).CombinedOutput()

	return []actions.Action{
		actions.NewLogAction(&action, fmt.Sprintf("Command output:\n%s", string(output))),
	}, err
}

func (s *ShellExecutor) isCommandAllowed(cmd string) bool {
	for _, pattern := range s.safePatterns {
		if pattern.MatchString(cmd) {
			return true
		}
	}
	return false
}
