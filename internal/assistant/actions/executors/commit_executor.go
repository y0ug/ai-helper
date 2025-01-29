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

	commit := action.Payload.(actions.CommitAction)
	hash, msg, err := e.repo.AutoCommit(ctx)
	logger.Debug("Committing changes", "message", msg, "hash", hash)
	if err := e.repo.Commit(commit.Message); err != nil {
		logger.Error("Commit failed", "error", err)
		return []actions.Action{
			actions.NewLogAction(&action, fmt.Sprintf("Commit failed: %v", err)),
		}, err
	}
	logger.Debug("Changes committed successfully")
	return []actions.Action{
		actions.NewLogAction(&action, "Changes committed successfully"),
	}, nil
}
