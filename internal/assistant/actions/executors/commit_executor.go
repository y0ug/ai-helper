package executors

import (
	"context"
	"fmt"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/repomanager"
)

// Create new commit_executor.go
type CommitExecutor struct {
	repo repomanager.RepoManagerInterface
}

func NewCommitExecutor(repo repomanager.RepoManagerInterface) *CommitExecutor {
	return &CommitExecutor{repo: repo}
}

func (e *CommitExecutor) CanHandle(action actions.Action) bool {
	_, ok := action.Payload.(actions.CommitAction)
	return ok
}

func (e *CommitExecutor) Handle(
	ctx context.Context,
	action actions.Action,
) ([]actions.Action, error) {
	logger := actions.GetLogger(ctx)

	_ = action.Payload.(actions.CommitAction)
	hash, msg, err := e.repo.AutoCommit(ctx)
	logger.Debug("Committing changes", "message", msg, "hash", hash)
	if err != nil {
		logger.Error("Commit failed", "error", err)
		return []actions.Action{
			actions.NewLogAction(&action, fmt.Sprintf("Commit failed: %v", err)),
		}, err
	}
	return []actions.Action{
		actions.NewLogAction(&action, fmt.Sprintf("Changes committed successfully %q")),
	}, nil
}
