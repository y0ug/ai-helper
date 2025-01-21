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
	StatusOutOfSync // New status for when file content differs from disk
)

func (s FileStatus) String() string {
	switch s {
	case StatusUnknown:
		return "unknown"
	case StatusUnmodified:
		return "unmodified"
	case StatusModified:
		return "modified"
	case StatusAdded:
		return "added"
	case StatusDeleted:
		return "deleted"
	case StatusRenamed:
		return "renamed"
	case StatusOutOfSync:
		return "out_of_sync"
	default:
		return "invalid"
	}
}

type FileInfo struct {
	Content    string
	Hash       string
	ReadOnly   bool
	LastUpdate time.Time
	GitStatus  FileStatus
	LastSent   string    // Stores the hash of the content when it was last sent
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
	MarkFileAsSent(path string) error
	HasFileChanged(path string) (bool, error)
}
