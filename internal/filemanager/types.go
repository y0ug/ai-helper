package filemanager

//go:generate go run go.uber.org/mock/mockgen@latest -destination=mock.go -package=filemanager .  FileManager

import (
	"time"
)

type FileStatus int

const (
	StatusUnknown FileStatus = iota
	StatusUnmodified
	StatusModified
	StatusAdded
	StatusDeleted
	StatusRenamed
)

type FileInfo struct {
	Content    string
	Hash       string
	ReadOnly   bool
	LastUpdate time.Time
	GitStatus  FileStatus
	// Can add more git-related fields like:
	Branch     string
	LastCommit string
}

type FileManager interface {
	AddFile(path string, readOnly bool) error
	RemoveFile(path string) error
	GetFileContent(path string) (string, bool, error)
	UpdateFileContent(path string, newContent string) error
	IsFileReadOnly(path string) (bool, error)
	GetFileLastUpdate(path string) (time.Time, error)
	GetFileStatus(path string) (FileStatus, error)
	GetFiles() map[string]*FileInfo
}
