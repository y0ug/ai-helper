package actions

import "github.com/y0ug/ai-helper/pkg/llmhaven/chat"

type ActionType string

func ToGeneric[T ActionPayload](a Action[T]) Action[any] {
	return Action[any]{
		Type:    a.Type,
		Payload: a.Payload,
	}
}

type Action[T any] struct {
	Type    ActionType `json:"type"`
	Payload T          `json:"payload,omitempty"`
}

type ActionPayload interface {
	Type() ActionType
}

func NewParsedAction[T ActionPayload](payload T) Action[any] {
	return Action[any]{
		Type:    payload.Type(),
		Payload: payload,
	}
}

type SendChatMessage struct {
	Msg        chat.ChatMessage `json:"msg"`
	NeedRender bool             `json:"need_render"`
}

func (SendChatMessage) Type() ActionType {
	return ActionType("send_chat_message")
}

type ApplyEdit struct {
	Filename string `json:"filename"`
	Original string `json:"original"`
	Updated  string `json:"updated"`
}

func (ApplyEdit) Type() ActionType {
	return ActionType("apply_edit")
}

type UserInputType string

var (
	UserInputTypeText    UserInputType = "text"
	UserInputTypeConfirm UserInputType = "confirm"
)

type AwaitUserInput struct {
	Question   string        `json:"question"`
	InputType  UserInputType `json:"input_type"`
	NextAction []Action[any] `json:"next_action"`
}

func (AwaitUserInput) Type() ActionType {
	return ActionType("await_user_input")
}

type ToolResultAction struct {
	ToolResult chat.MessageContent `json:"tool_result"`
	NextAction []Action[any]       `json:"next_action"`
}

func (ToolResultAction) Type() ActionType {
	return ActionType("tool_result")
}

type ShellCommand struct {
	Command string `json:"command"`
}

func (ShellCommand) Type() ActionType {
	return ActionType("shell_command")
}
