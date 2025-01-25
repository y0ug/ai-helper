package actions

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type ActionType string

const (
	ActionTypeEdit         ActionType = "edit"
	ActionTypeShellCommand ActionType = "shell_command"
	ActionTypeUserConfirm  ActionType = "user_confirm"
	ActionTypeCommit       ActionType = "commit"
	ActionTypeLog          ActionType = "log"
	ActionTypeLLMRequest   ActionType = "llm_request"
	ActionTypeLLMResponse  ActionType = "llm_response"
	ActionTypeAddMessage   ActionType = "add_message"
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

func (a *Action) IsToolCall() bool {
	return a.Context.ToolCallID != ""
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
	return NewAction(ActionTypeShellCommand, ShellCommandAction{
		Command:      s.Command,
		NeedsConfirm: false, // No longer needs confirmation
		Confirmed:    true,
		Output:       s.Output,
	}).WithParent(parentAction)
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

func (a Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(a.Type)),
		slog.String("action_id", a.ID.String()),
		slog.String("chain_id", a.Context.ChainID.String()),
		slog.String("parent_id", a.Context.ParentID.String()),
		slog.String("tool_call_id", a.Context.ToolCallID),
		slog.String("created_at", a.Context.CreatedAt.Format(time.RFC3339)),
	)
}

// New helper method to maintain chain IDs

func (a Action) WithToolCallID(id string) Action {
	a.Context.ToolCallID = id
	return a
}

func (a Action) WithParent(parent *Action) Action {
	if parent != nil {
		a.Context.ChainID = parent.Context.ChainID
		a.Context.ParentID = parent.ID
		// TODO: check if we want this
		// we only copy the tool call ID if the parent is a tool call
		if parent.IsToolCall() {
			a.Context.ToolCallID = parent.Context.ToolCallID
		}
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
	return Action{
		ID:      uuid.New(),
		Type:    actionType,
		Payload: payload,
		Context: ActionContext{
			ChainID:   uuid.New(),
			CreatedAt: time.Now(),
		},
	}
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
	return NewAction(ActionTypeEdit, ApplyEdit{
		Filename: filename,
		Original: original,
		Updated:  updated,
	}).WithParent(parentAction)
}

func NewShellCommand(parentAction *Action, command string, needsConfirm bool) Action {
	return NewAction(ActionTypeShellCommand, ShellCommandAction{
		Command:      command,
		NeedsConfirm: needsConfirm,
	}).WithParent(parentAction)
}

func NewUserConfirmAction(parentAction *Action, question string) Action {
	return NewAction(ActionTypeUserConfirm, UserConfirmAction{
		Question: question,
		Context:  parentAction.Context,
	}).WithParent(parentAction)
}

func NewCommitAction(parentAction *Action, message string) Action {
	return NewAction(ActionTypeCommit, CommitAction{
		Message: message,
	}).WithParent(parentAction)
}

func NewLogAction(parentAction *Action, message string) Action {
	return NewAction(ActionTypeLog, LogAction{
		Message: message,
	}).WithParent(parentAction)
}

type LLMRequestAction struct {
	Prompt string
}
type LLMResponseAction struct {
	Response string
}

type AddMessageAction struct {
	Msg chat.ChatMessage
}

func NewLLMRequestAction(prompt string) Action {
	return NewAction(ActionTypeLLMRequest, LLMRequestAction{Prompt: prompt})
}

func NewLLMResponseAction(response string) Action {
	return NewAction(ActionTypeLLMResponse, LLMResponseAction{Response: response})
}

func NewAddMessageAction(msg chat.ChatMessage) Action {
	return NewAction(ActionTypeAddMessage, AddMessageAction{Msg: msg})
}
