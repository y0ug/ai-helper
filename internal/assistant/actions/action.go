package actions

import (
	"time"

	"github.com/google/uuid"
)

type ActionType string

const (
	ActionTypeEdit         ActionType = "edit"
	ActionTypeShellCommand ActionType = "shell_command"
	ActionTypeUserConfirm  ActionType = "user_confirm"
	ActionTypeCommit       ActionType = "commit"
	ActionTypeLog          ActionType = "log"
)

type ActionContext struct {
	ToolCallID string
	ParentID   uuid.UUID
	ChainID    uuid.UUID
	CreatedAt  time.Time
}

type Action struct {
	ID      uuid.UUID
	Type    ActionType
	Payload interface{}
	Context ActionContext
}

type ApplyEdit struct {
	Filename string
	Original string
	Updated  string
}

type ShellCommandAction struct {
	Command      string
	NeedsConfirm bool
	Confirmed    bool
	Output       string
}

func (s *ShellCommandAction) WithConfirmed(parentAction *Action) Action {
	return NewActionWithParent(ActionTypeShellCommand, ShellCommandAction{
		Command:      s.Command,
		NeedsConfirm: s.NeedsConfirm,
		Confirmed:    true,
		Output:       s.Output,
	}, parentAction)
}

type UserConfirmAction struct {
	Question string
	Context  ActionContext
}

type CommitAction struct {
	Message string
}

type LogAction struct {
	Message string
}

type UserResponseAction struct {
	Allowed bool
	Context ActionContext
}

// Helper to convert an action to an Array of actions
func Slice(action ...Action) []Action {
	return action
}

func NewAction(actionType ActionType, payload interface{}) Action {
	return NewActionWithParent(actionType, payload, nil)
}

func NewActionWithParent(actionType ActionType, payload interface{}, parentAction *Action) Action {
	a := Action{
		ID:      uuid.New(),
		Type:    actionType,
		Payload: payload,
	}
	if parentAction != nil {
		a.Context = ActionContext{
			ParentID:   parentAction.ID,
			ChainID:    parentAction.Context.ChainID,
			ToolCallID: parentAction.Context.ToolCallID,
			CreatedAt:  time.Now(),
		}
	}
	return a
}

func NewApplyEdit(parentAction *Action, filename, original, updated string) Action {
	return NewActionWithParent(ActionTypeEdit, ApplyEdit{
		Filename: filename,
		Original: original,
		Updated:  updated,
	}, parentAction)
}

func NewShellCommand(parentAction *Action, command string, needsConfirm bool) Action {
	return NewActionWithParent(ActionTypeShellCommand, ShellCommandAction{
		Command:      command,
		NeedsConfirm: needsConfirm,
	}, parentAction)
}

func NewUserConfirmAction(parentAction *Action, question string) Action {
	return NewActionWithParent(ActionTypeUserConfirm, UserConfirmAction{
		Question: question,
		Context:  parentAction.Context,
	}, parentAction)
}

func NewCommitAction(parentAction *Action, message string) Action {
	return NewActionWithParent(ActionTypeCommit, CommitAction{
		Message: message,
	}, parentAction)
}

func NewLogAction(parentAction *Action, message string) Action {
	return NewActionWithParent(ActionTypeLog, LogAction{
		Message: message,
	}, parentAction)
}
