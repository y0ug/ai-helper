package actions

import (
	"fmt"
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
	ID        uuid.UUID
	Type      ActionType
	Payload   interface{}
	Context   ActionContext
	Completed bool
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
	Executed     bool // Add execution state
	Success      bool // Add success stat
}

//	func (s *ShellCommandAction) WithConfirmed(parentAction *Action) Action {
//		return NewActionWithParent(ActionTypeShellCommand, ShellCommandAction{
//			Command:      s.Command,
//			NeedsConfirm: s.NeedsConfirm,
//			Confirmed:    true,
//			Output:       s.Output,
//		}, parentAction)
//	}
func (s *ShellCommandAction) WithConfirmed(parentAction *Action) Action {
	return NewActionWithParent(ActionTypeShellCommand, ShellCommandAction{
		Command:      s.Command,
		NeedsConfirm: false, // No longer needs confirmation
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

// In actions/action.go
func (a Action) String() string {
	// Add indication of completion status
	status := ""
	if a.Completed {
		status = "✓ "
	}
	switch v := a.Payload.(type) {
	case ApplyEdit:
		return fmt.Sprintf("%sEDIT %s: %q → %q", status,
			v.Filename, shorten(v.Original), shorten(v.Updated))
	case CommitAction:
		return fmt.Sprintf("%sCOMMIT: %s", status, v.Message)
	case ShellCommandAction:
		return fmt.Sprintf("%sCMD: %s (confirmed:%v)", status,
			v.Command, v.Confirmed)
	case LogAction:
		return fmt.Sprintf("%sLOG: %s", status, v.Message)
	default:
		return fmt.Sprintf("%sACTION-%s", status, a.Type)
	}
}

// func (a Action) String() string {
// 	switch payload := a.Payload.(type) {
// 	case ApplyEdit:
// 		return fmt.Sprintf("Edit %s: %d chars replaced",
// 			payload.Filename,
// 			len(payload.Updated))
// 	case ShellCommandAction:
// 		return fmt.Sprintf("Command executed: %s (confirmed: %v)",
// 			payload.Command,
// 			payload.Confirmed)
// 	case LogAction:
// 		return fmt.Sprintf("LOG: %s", payload.Message)
// 	default:
// 		return fmt.Sprintf("Action %s (%s)", a.Type, a.ID)
// 	}
// }

// In actions/action.go
// func (a Action) String() string {
// 	switch v := a.Payload.(type) {
// 	case ApplyEdit:
// 		return fmt.Sprintf("EDIT %s: %q → %q",
// 			v.Filename,
// 			shorten(v.Original),
// 			shorten(v.Updated))
// 	case CommitAction:
// 		return fmt.Sprintf("COMMIT: %s", v.Message)
// 	case ShellCommandAction:
// 		return fmt.Sprintf("CMD: %s (confirmed:%v)",
// 			v.Command, v.Confirmed)
// 	case LogAction:
// 		return fmt.Sprintf("LOG: %s", v.Message)
// 	default:
// 		return fmt.Sprintf("ACTION-%s", a.Type)
// 	}
// }

// New helper method to maintain chain IDs
func (a Action) WithChainID(parent *Action) Action {
	if parent != nil {
		a.Context.ChainID = parent.Context.ChainID
		a.Context.ParentID = parent.ID
	}
	return a
}

func shorten(s string) string {
	if len(s) > 20 {
		return s[:17] + "..."
	}
	return s
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
	} else {
		a.Context = ActionContext{
			ChainID:   uuid.New(),
			CreatedAt: time.Now(),
		}
	}
	return a
}

func NewApplyEdit(parentAction *Action, filename, original, updated string) Action {
	return NewActionWithParent(ActionTypeEdit, ApplyEdit{
		Filename: filename,
		Original: original,
		Updated:  updated,
	}, parentAction).WithChainID(parentAction) // Add this method
}

// func NewApplyEdit(parentAction *Action, filename, original, updated string) Action {
// 	return NewActionWithParent(ActionTypeEdit, ApplyEdit{
// 		Filename: filename,
// 		Original: original,
// 		Updated:  updated,
// 	}, parentAction)
// }

func NewShellCommand(parentAction *Action, command string, needsConfirm bool) Action {
	return NewActionWithParent(ActionTypeShellCommand, ShellCommandAction{
		Command:      command,
		NeedsConfirm: needsConfirm,
	}, parentAction).WithChainID(parentAction)
}

func NewUserConfirmAction(parentAction *Action, question string) Action {
	return NewActionWithParent(ActionTypeUserConfirm, UserConfirmAction{
		Question: question,
		Context:  parentAction.Context,
	}, parentAction).WithChainID(parentAction)
}

func NewCommitAction(parentAction *Action, message string) Action {
	return NewActionWithParent(ActionTypeCommit, CommitAction{
		Message: message,
	}, parentAction).WithChainID(parentAction)
}

func NewLogAction(parentAction *Action, message string) Action {
	return NewActionWithParent(ActionTypeLog, LogAction{
		Message: message,
	}, parentAction).WithChainID(parentAction)
}
