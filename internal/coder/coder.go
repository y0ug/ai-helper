package coder

import (
	"github.com/y0ug/ai-helper/internal/coder/models"
	"github.com/y0ug/ai-helper/internal/coder/prompts"
)

type Coder interface {
	GetEdits() ([]Edit, error)
	ApplyEdits([]Edit) error
	Run(message string) error
	GetPrompts() *prompts.BasePrompts
	// ... other common methods
}

type CoderOptions struct {
	MainModel  *models.ModelSettings
	EditFormat string
	// IO         *io.InputOutput
	// ... other options
}

type FileManager interface {
	ReadFile(path string) (string, error)
	WriteFile(path string, content string) error
	IsFileEditable(path string) bool
}

type GitManager interface {
	CommitChanges(files []string, message string) (string, error)
	GetHeadCommitSHA() string
	IsGitIgnored(path string) bool
}

type LLMClient interface {
	Complete(messages []prompts.Message, opts CompleteOptions) (string, error)
}

type CompleteOptions struct {
	Temperature float64
	MaxTokens   int
	Stream      bool
}

// IOInterface defines methods for IO operations
type IOInterface interface {
	ToolError(msg string)
	ToolOutput(msg string, bold bool)
	ToolWarning(msg string)
}
