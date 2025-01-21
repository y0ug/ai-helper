package gitrepo

import (
	"time"

	"github.com/go-git/go-git/v5"
)

//go:generate go run go.uber.org/mock/mockgen@latest -destination=mock.go -package=gitrepo . GitRepoInterface,IOInterface

// GitRepoInterface defines the contract for git repository operations
type GitRepoInterface interface {
	Commit(fnames []string, context string, message string, aiderEdits bool) (string, string, error)
	GetDiffs(fnames []string) (string, error)
	DiffCommits(pretty bool, fromCommit, toCommit string) (string, error)
	GetTrackedFiles() ([]string, error)
	IsIgnoredFile(fname string) bool
	PathInRepo(path string) bool
	GetDirtyFiles() ([]string, error)
	IsDirty(path string) bool
	GetHeadCommit() (string, error)
	GetHeadCommitSHA(short bool) (string, error)
	GetHeadCommitMessage(defaultMsg string) (string, error)
}

// GitRepo implements GitRepoInterface
type GitRepo struct {
	io              IOInterface
	repo            *git.Repository
	root            string
	normalizedPath  map[string]string
	treeFiles       map[string]map[string]struct{}
	ignoreFileCache map[string]bool

	aiderIgnoreFile string
	aiderIgnoreSpec *string //[]gitignore.Pattern
	aiderIgnoreTS   time.Time
	lastIgnoreCheck time.Time

	subtreeOnly                     bool
	attributeAuthor                 bool
	attributeCommitter              bool
	attributeCommitMessageAuthor    bool
	attributeCommitMessageCommitter bool
	commitPrompt                    string
}

// IOInterface defines methods for IO operations
type IOInterface interface {
	ToolError(msg string)
	ToolOutput(msg string, bold bool)
	ToolWarning(msg string)
}

// Config holds the configuration for creating a new GitRepo
type Config struct {
	IO                              IOInterface
	Fnames                          []string
	GitDname                        string
	AiderIgnoreFile                 string
	AttributeAuthor                 bool
	AttributeCommitter              bool
	AttributeCommitMessageAuthor    bool
	AttributeCommitMessageCommitter bool
	CommitPrompt                    string
	SubtreeOnly                     bool
}
