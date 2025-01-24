package responseextractor

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type (
	ExtractorType string
	ActionType    string
)

type ExtractorError struct {
	Name          string
	ExtractorType string
	Err           error
	Line          int
}

func (e *ExtractorError) Error() string {
	return fmt.Sprintf("%s (line %d) - %s %+w", e.Name, e.Line, e.ExtractorType, e.Err)
}

func NewExtractorError(
	name string,
	extractorType ExtractorType,
	err error,
	line int,
) *ExtractorError {
	return &ExtractorError{
		Name:          name,
		ExtractorType: string(extractorType),
		Err:           err,
		Line:          line,
	}
}

type ParsedAction struct {
	Type  ActionType      // "apply_edit", "shell_cmd", "ask_user", etc. Should match a tool command that will be executed after
	Input json.RawMessage // JSON object with details specific to the action type
}

// tool_result is just an array of ChatContent with a prefill tool_result type
const (
	ActionApplyError          ActionType = "error"
	ActionApplyEdit           ActionType = "apply_edit"
	ActionApplyEditToolResult ActionType = "tool_result"
)

type Edit struct {
	Filename string
	Original string
	Updated  string
}

const ActionShellCmd ActionType = "shell_cmd"

type ShellCommand struct {
	Command string
}

const (
	ExtractorEditDiff       ExtractorType = "editblock-diff"
	ExtractorEditDiffFenced ExtractorType = "editblock-diff-fenced"
	ExtractorEditFuncWhole  ExtractorType = "edit-func-whole"
)

type ResponseExtractor interface {
	Extract(msg *chat.ChatMessage) ([]ParsedAction, error)
	GetChatTools() []chat.Tool
	SetFence(Fence)
	GetName() string
	GetFormat() ExtractorType
}

func New(editFormat ExtractorType, logger *slog.Logger) ResponseExtractor {
	fence := DefaultFences[0]
	switch editFormat {
	case ExtractorEditDiff:
		return NewBlockExtractor(logger, editFormat, fence)
	case ExtractorEditDiffFenced:
		return NewBlockExtractor(logger, editFormat, fence)
	case ExtractorEditFuncWhole:
		return NewFuncWholeExtractor(logger, editFormat, fence)
	default:
		return nil
	}
}

type Fence [2]string

// DefaultFences defines all possible fencing options in order of preference
var DefaultFences = []Fence{
	{"```", "```"},
	{"````", "````"},
	{"<source>", "</source>"},
	{"<code>", "</code>"},
	{"<pre>", "</pre>"},
	{"<codeblock>", "</codeblock>"},
	{"<sourcecode>", "</sourcecode>"},
}
