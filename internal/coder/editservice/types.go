package editservice

import (
	"log/slog"

	"github.com/y0ug/ai-helper/internal/filemanager"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type (
	EditMode   string
	EditFormat string
)

const (
	EditFormatDiff        EditFormat = "diff"
	EditFormatDiffFenced  EditFormat = "diff-fenced"
	EditFormatArchitect   EditFormat = "architect"
	EditFormatAsk         EditFormat = "ask"
	EditFormatCode        EditFormat = "code"
	EditFormatEditorDiff  EditFormat = "editor-diff"
	EditFormatEditorWhole EditFormat = "editor-whole"
	EditFormatHelp        EditFormat = "help"
	EditFormatFunc        EditFormat = "func"
	EditFormatUdiff       EditFormat = "udiff"
	EditFormatWhole       EditFormat = "whole"
)

type EditService interface {
	GetEdits(content string) []EditResult
	GetEditsMsg(msg *chat.ChatMessage) ([]EditResult, []chat.MessageContent)
	GetChatTools() []chat.Tool
	ApplyEdits(filemanager.FileManager, []Edit, bool) error
	SetFence(Fence)
	GetName() string
	GetFormat() EditFormat
}

func New(editFormat EditFormat, logger *slog.Logger) EditService {
	fence := DefaultFences[0]
	switch editFormat {
	case EditFormatDiff:
		return NewEditBlockService(logger, editFormat, fence)
	case EditFormatDiffFenced:
		return NewEditBlockService(logger, editFormat, fence)
	case EditFormatArchitect:
		return NewNoEditService(logger, editFormat)
	case EditFormatAsk:
		return NewNoEditService(logger, editFormat)
	case EditFormatCode:
		return NewNoEditService(logger, editFormat)
	case EditFormatEditorDiff:
		return nil
	case EditFormatEditorWhole:
		return nil
	case EditFormatHelp:
		return NewNoEditService(logger, editFormat)
	case EditFormatFunc:
		return NewSingleWholeFileFuncService(logger, editFormat, fence)
	case EditFormatUdiff:
		return nil
	case EditFormatWhole:
		return NewWholeFileService(logger, editFormat, fence)
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

type Edit struct {
	Filename string
	Original string
	Updated  string
	Format   EditFormat
	Mode     EditMode
}

type ShellCommand struct {
	Command string
}

type EditResult struct {
	Edit  *Edit
	Shell *ShellCommand
	Err   error
}
