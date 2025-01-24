package extractor

import (
	"fmt"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
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

const (
	ExtractorEditDiff       ExtractorType = "editblock-diff"
	ExtractorEditDiffFenced ExtractorType = "editblock-diff-fenced"
	ExtractorEditFuncWhole  ExtractorType = "edit-func-whole"
)

type ResponseExtractor interface {
	Extract(msg *chat.ChatMessage) ([]actions.Action[any], error)
	GetChatTools() []chat.Tool
	SetFence(Fence)
	GetName() string
	GetFormat() ExtractorType
}

func New(editFormat ExtractorType, logger *slog.Logger) ResponseExtractor {
	fence := DefaultFences[0]
	switch editFormat {
	case ExtractorEditDiff:
		return NewEditBlockExtractor(logger, editFormat, fence)
	case ExtractorEditDiffFenced:
		return NewEditBlockExtractor(logger, editFormat, fence)
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
