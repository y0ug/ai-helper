package executors

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/repomanager"
	"github.com/y0ug/ai-helper/internal/assistant/validation"
)

type EditExecutor struct {
	repo      repomanager.RepoManagerInterface
	validator validation.Validator
	logger    *slog.Logger
}

func NewEditExecutor(
	repo repomanager.RepoManagerInterface,
	validator validation.Validator,
	logger *slog.Logger,
) *EditExecutor {
	return &EditExecutor{
		repo:      repo,
		validator: validator,
		logger:    logger,
	}
}

func (e *EditExecutor) CanHandle(action actions.Action) bool {
	_, ok := action.Payload.(actions.ApplyEdit)
	return ok
}

func (e *EditExecutor) Handle(
	ctx context.Context,
	action actions.Action,
) ([]actions.Action, error) {
	edit := action.Payload.(actions.ApplyEdit)

	// Run validation pipeline
	validationResult := e.validator.Validate(ctx, edit.Filename, edit.Original, edit.Updated)

	if !validationResult.Valid {
		var followUps []actions.Action
		for _, issue := range validationResult.Issues {
			followUps = append(followUps, actions.NewLogAction(&action,
				fmt.Sprintf("Validation %s: %s (line %d)",
					strings.ToLower(issue.Level.String()),
					issue.Message,
					issue.Line,
				),
			))
		}
		followUps = append(
			followUps,
			actions.NewLogAction(&action, "Edit rejected due to validation errors"),
		)
		return followUps, fmt.Errorf("validation failed")
	}

	// Proceed with applying edit
	if err := e.repo.ApplyEdit(edit); err != nil {
		// if err := repo.ApplyEdit(edit); err != nil {
		return []actions.Action{
			actions.NewLogAction(&action, fmt.Sprintf("Edit failed: %v", err)),
		}, fmt.Errorf("failed to apply edit: %w", err)
	}

	return []actions.Action{
		actions.NewCommitAction(&action, fmt.Sprintf("Applied edit to %s", edit.Filename)),
		actions.NewLogAction(&action, "Edit applied successfully"),
	}, nil
}
