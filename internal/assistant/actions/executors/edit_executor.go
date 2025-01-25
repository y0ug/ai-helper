package executors

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/repomanager"
	"github.com/y0ug/ai-helper/internal/assistant/validation"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
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

func NewActionAddMsgToolResult(toolCallID string, content string) actions.Action {
	return actions.NewAddMessageAction(
		*chat.NewMessage("tool",
			chat.NewToolResultContent(
				toolCallID,
				content,
			)))
}

func (e *EditExecutor) Handle(
	ctx context.Context,
	action actions.Action,
) ([]actions.Action, error) {
	edit := action.Payload.(actions.ApplyEdit)
	var followUps []actions.Action

	// Run validation pipeline
	validationResult := e.validator.Validate(ctx, edit.Filename, edit.Original, edit.Updated)

	if !validationResult.Valid {
		for _, issue := range validationResult.Issues {
			followUps = append(followUps, actions.NewLogAction(&action,
				fmt.Sprintf("Validation %s: %s (line %d)",
					strings.ToLower(issue.Level.String()),
					issue.Message,
					issue.Line,
				),
			))
		}
		if action.IsToolCall() {
			followUps = append(
				followUps,
				NewActionAddMsgToolResult(
					action.Context.ToolCallID,
					"Edit rejected due to validation errors",
				).WithParent(&action),
			)
		}
		followUps = append(
			followUps,
			actions.NewLogAction(&action, "Edit rejected due to validation errors"),
		)
		return followUps, fmt.Errorf("validation failed")
	}

	// Proceed with applying edit
	if err := e.repo.ApplyEdit(edit); err != nil {
		if action.IsToolCall() {
			followUps = append(
				followUps,
				NewActionAddMsgToolResult(
					action.Context.ToolCallID,
					"failed to apply edit",
				).WithParent(&action),
			)
		}

		followUps = append(
			followUps,
			actions.NewLogAction(&action, fmt.Sprintf("Edit failed: %v", err)),
		)
		// if err := repo.ApplyEdit(edit); err != nil {
		return followUps, fmt.Errorf("failed to apply edit: %w", err)
	}

	if action.IsToolCall() {
		followUps = append(
			followUps,
			NewActionAddMsgToolResult(
				action.Context.ToolCallID,
				"Edit applied successfully",
			).WithParent(&action),
		)
	}
	followUps = append(
		followUps,
		actions.NewCommitAction(&action, fmt.Sprintf("Applied edit to %s", edit.Filename)),
		actions.NewLogAction(&action, "Edit applied successfully"))
	return followUps, nil
}
