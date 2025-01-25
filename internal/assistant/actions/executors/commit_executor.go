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
	commit := action.Payload.(actions.CommitAction)
	if err := e.repo.GetFM().Commit(commit.Message); err != nil {
		return []actions.Action{
			actions.NewLogAction(&action, fmt.Sprintf("Commit failed: %v", err)),
		}, err
	}
	return []actions.Action{
		actions.NewLogAction(&action, "Changes committed successfully"),
	}, nil
}
