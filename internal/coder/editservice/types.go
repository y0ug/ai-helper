package editservice

import "github.com/y0ug/ai-helper/internal/filemanager"

type (
	EditMode   string
	EditFormat string
)

const (
	EditBlockMode  EditMode   = "block"
	EditWholeMode  EditMode   = "whole"
	EditFormatDiff EditFormat = "diff"
)

type EditService interface {
	GetEdits(content string) []EditResult
	ApplyEdits(filemanager.FileManager, []Edit, bool) error
	SetFence(Fence)
}

type Fence [2]string

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
