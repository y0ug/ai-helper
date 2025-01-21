package gitrepo

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func New(config Config) (*GitRepo, error) {
	gr := &GitRepo{
		io:              config.IO,
		normalizedPath:  make(map[string]string),
		treeFiles:       make(map[string]map[string]struct{}),
		ignoreFileCache: make(map[string]bool),

		attributeAuthor:                 config.AttributeAuthor,
		attributeCommitter:              config.AttributeCommitter,
		attributeCommitMessageAuthor:    config.AttributeCommitMessageAuthor,
		attributeCommitMessageCommitter: config.AttributeCommitMessageCommitter,
		commitPrompt:                    config.CommitPrompt,
		subtreeOnly:                     config.SubtreeOnly,
	}

	repoPath, err := gr.findGitRepo(config.Fnames, config.GitDname)
	if err != nil {
		return nil, err
	}

	gr.repo, err = git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}

	gr.root = repoPath

	if config.AiderIgnoreFile != "" {
		gr.aiderIgnoreFile = config.AiderIgnoreFile
	}

	return gr, nil
}

func (gr *GitRepo) Commit(
	fnames []string,
	context string,
	message string,
	aiderEdits bool,
) (string, string, error) {
	if len(fnames) == 0 && !gr.IsDirty("") {
		return "", "", nil
	}

	diffs, err := gr.GetDiffs(fnames)
	if err != nil {
		return "", "", err
	}

	if diffs == "" {
		return "", "", nil
	}

	commitMsg := message
	if commitMsg == "" {
		commitMsg = gr.generateCommitMessage(diffs, context)
	}

	if aiderEdits && gr.attributeCommitMessageAuthor {
		commitMsg = "aider: " + commitMsg
	} else if gr.attributeCommitMessageCommitter {
		commitMsg = "aider: " + commitMsg
	}

	if commitMsg == "" {
		commitMsg = "(no commit message provided)"
	}

	w, err := gr.repo.Worktree()
	if err != nil {
		return "", "", err
	}

	// Add files to staging
	for _, fname := range fnames {
		_, err := w.Add(fname)
		if err != nil {
			gr.io.ToolError(fmt.Sprintf("Unable to add %s: %v", fname, err))
			continue
		}
	}

	// Prepare commit options
	opts := &git.CommitOptions{
		Author: &object.Signature{
			Name:  gr.getCommitterName(aiderEdits),
			Email: "aider@aider.chat",
			When:  time.Now(),
		},
	}

	// Commit
	hash, err := w.Commit(commitMsg, opts)
	if err != nil {
		return "", "", err
	}

	shortHash := hash.String()[:7]
	gr.io.ToolOutput(fmt.Sprintf("Commit %s %s", shortHash, commitMsg), true)

	return shortHash, commitMsg, nil
}

func (gr *GitRepo) PathInRepo(path string) bool {
	if path == "" {
		return false
	}

	trackedFiles, err := gr.GetTrackedFiles()
	if err != nil {
		return false
	}

	normalizedPath := gr.normalizePath(path)
	for _, tracked := range trackedFiles {
		if tracked == normalizedPath {
			return true
		}
	}
	return false
}

func (gr *GitRepo) GetDiffs(fnames []string) (string, error) {
	w, err := gr.repo.Worktree()
	if err != nil {
		return "", err
	}

	status, err := w.Status()
	if err != nil {
		return "", err
	}

	var diffs strings.Builder

	// Handle new files first
	for _, fname := range fnames {
		if !gr.PathInRepo(fname) {
			diffs.WriteString(fmt.Sprintf("Added %s\n", fname))
		}
	}

	// Get current branch's head commit
	head, err := gr.repo.Head()
	if err != nil && err != plumbing.ErrReferenceNotFound {
		return "", err
	}

	// If we have a head commit, diff against it
	if head != nil {
		commit, err := gr.repo.CommitObject(head.Hash())
		if err != nil {
			return "", err
		}

		tree, err := commit.Tree()
		if err != nil {
			return "", err
		}

		changes, err := tree.Diff(w.)
		if err != nil {
			return "", err
		}

		for _, change := range changes {
			patch, err := change.Patch()
			if err != nil {
				return "", err
			}
			diffs.WriteString(patch.String())
		}
	} else {
		// For new repositories without commits
		for fname := range status {
			if len(fnames) > 0 && !contains(fnames, fname) {
				continue
			}
			file, err := w.Filesystem.Open(fname)
			if err != nil {
				continue
			}
			content, err := io.ReadAll(file)
			file.Close()
			if err != nil {
				continue
			}
			diffs.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", fname, fname))
			diffs.WriteString(fmt.Sprintf("new file mode 100644\n"))
			diffs.WriteString(fmt.Sprintf("--- /dev/null\n"))
			diffs.WriteString(fmt.Sprintf("+++ b/%s\n", fname))
			diffs.WriteString("@@ -0,0 +1,1 @@\n")
			diffs.WriteString("+" + string(content))
		}
	}

	return diffs.String(), nil
}

// Helper function to create a simple diff output
func createSimpleDiff(original, current string) string {
	originalLines := strings.Split(original, "\n")
	currentLines := strings.Split(current, "\n")

	var diff strings.Builder
	diff.WriteString(fmt.Sprintf("@@ -1,%d +1,%d @@\n", len(originalLines), len(currentLines)))

	for _, line := range originalLines {
		diff.WriteString("-" + line + "\n")
	}
	for _, line := range currentLines {
		diff.WriteString("+" + line + "\n")
	}

	return diff.String()
}

// Update the DiffCommits method to use the correct plumbing.Hash
func (gr *GitRepo) DiffCommits(pretty bool, fromCommit, toCommit string) (string, error) {
	from, err := gr.repo.CommitObject(plumbing.NewHash(fromCommit))
	if err != nil {
		return "", err
	}

	to, err := gr.repo.CommitObject(plumbing.NewHash(toCommit))
	if err != nil {
		return "", err
	}

	// Get the trees from both commits
	fromTree, err := from.Tree()
	if err != nil {
		return "", err
	}

	toTree, err := to.Tree()
	if err != nil {
		return "", err
	}

	// Calculate the changes between trees
	changes, err := object.DiffTree(fromTree, toTree)
	if err != nil {
		return "", err
	}

	var diff strings.Builder
	for _, change := range changes {
		patch, err := change.Patch()
		if err != nil {
			continue
		}
		diff.WriteString(patch.String())
	}

	return diff.String(), nil
}

// Update IsIgnoredFile to use the correct gitignore API
func (gr *GitRepo) IsIgnoredFile(fname string) bool {
	if cached, ok := gr.ignoreFileCache[fname]; ok {
		return cached
	}

	w, err := gr.repo.Worktree()
	if err != nil {
		return false
	}

	// Check if the file matches any gitignore patterns
	excluded := false
	patterns, err := gitignore.ReadPatterns(w.Filesystem, nil)
	if err == nil {
		matcher := gitignore.NewMatcher(patterns)
		excluded = matcher.Match(strings.Split(fname, "/"), false)
	}

	// Check aider ignore file if configured
	if gr.aiderIgnoreFile != "" && gr.checkAiderIgnore(fname) {
		excluded = true
	}

	gr.ignoreFileCache[fname] = excluded
	return excluded
}

func (gr *GitRepo) GetTrackedFiles() ([]string, error) {
	head, err := gr.repo.Head()
	if err != nil && err != plumbing.ErrReferenceNotFound {
		return nil, err
	}

	files := make(map[string]struct{})

	if head != nil {
		commit, err := gr.repo.CommitObject(head.Hash())
		if err != nil {
			return nil, err
		}

		tree, err := commit.Tree()
		if err != nil {
			return nil, err
		}

		tree.Files().ForEach(func(f *object.File) error {
			if !gr.IsIgnoredFile(f.Name) {
				files[gr.normalizePath(f.Name)] = struct{}{}
			}
			return nil
		})
	}

	// Add staged files
	w, err := gr.repo.Worktree()
	if err != nil {
		return nil, err
	}

	status, err := w.Status()
	if err != nil {
		return nil, err
	}

	for fname := range status {
		if !gr.IsIgnoredFile(fname) {
			files[gr.normalizePath(fname)] = struct{}{}
		}
	}

	result := make([]string, 0, len(files))
	for fname := range files {
		result = append(result, fname)
	}

	return result, nil
}

func (gr *GitRepo) GetDirtyFiles() ([]string, error) {
	w, err := gr.repo.Worktree()
	if err != nil {
		return nil, err
	}

	status, err := w.Status()
	if err != nil {
		return nil, err
	}

	dirtyFiles := make([]string, 0)
	for fname, fileStatus := range status {
		if fileStatus.Staging != git.Unmodified || fileStatus.Worktree != git.Unmodified {
			dirtyFiles = append(dirtyFiles, fname)
		}
	}

	return dirtyFiles, nil
}

func (gr *GitRepo) IsDirty(path string) bool {
	if path != "" && !gr.PathInRepo(path) {
		return true
	}

	w, err := gr.repo.Worktree()
	if err != nil {
		return false
	}

	status, err := w.Status()
	if err != nil {
		return false
	}

	if path == "" {
		return !status.IsClean()
	}

	fileStatus := status.File(path)

	return fileStatus.Staging != git.Unmodified || fileStatus.Worktree != git.Unmodified
}

func (gr *GitRepo) GetHeadCommit() (string, error) {
	head, err := gr.repo.Head()
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return "", nil
		}
		return "", err
	}
	return head.Hash().String(), nil
}

func (gr *GitRepo) GetHeadCommitSHA(short bool) (string, error) {
	hash, err := gr.GetHeadCommit()
	if err != nil {
		return "", err
	}
	if hash == "" {
		return "", nil
	}
	if short {
		return hash[:7], nil
	}
	return hash, nil
}

func (gr *GitRepo) GetHeadCommitMessage(defaultMsg string) (string, error) {
	head, err := gr.repo.Head()
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return defaultMsg, nil
		}
		return "", err
	}

	commit, err := gr.repo.CommitObject(head.Hash())
	if err != nil {
		return "", err
	}

	return commit.Message, nil
}

// Helper functions

func (gr *GitRepo) normalizePath(path string) string {
	if normalized, ok := gr.normalizedPath[path]; ok {
		return normalized
	}

	normalized := filepath.ToSlash(filepath.Clean(path))
	gr.normalizedPath[path] = normalized
	return normalized
}

func (gr *GitRepo) checkAiderIgnore(fname string) bool {
	// Check if it's time to refresh the ignore patterns
	now := time.Now()
	if now.Sub(gr.lastIgnoreCheck) >= time.Second {
		gr.refreshAiderIgnore()
		gr.lastIgnoreCheck = now
	}

	return false
	// if gr.aiderIgnoreSpec == nil {
	// 	return false
	// }
	//
	// normalizedPath := gr.normalizePath(fname)
	// return gr.aiderIgnoreSpec.Match(normalizedPath)
}

func (gr *GitRepo) refreshAiderIgnore() {
	if gr.aiderIgnoreFile == "" {
		return
	}

	info, err := os.Stat(gr.aiderIgnoreFile)
	if err != nil {
		return
	}

	if !info.ModTime().After(gr.aiderIgnoreTS) {
		return
	}

	content, err := os.ReadFile(gr.aiderIgnoreFile)
	if err != nil {
		return
	}

	patterns := strings.Split(string(content), "\n")
	var validPatterns []string
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" && !strings.HasPrefix(pattern, "#") {
			validPatterns = append(validPatterns, pattern)
		}
	}

	// gr.aiderIgnoreSpec, _ = gitignore.NewMatcher(strings.Join(validPatterns, "|"))
	gr.aiderIgnoreTS = info.ModTime()
	gr.ignoreFileCache = make(map[string]bool)
}

func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func (gr *GitRepo) findGitRepo(fnames []string, gitDname string) (string, error) {
	if gitDname != "" {
		return gitDname, nil
	}

	checkPaths := []string{"."}
	if len(fnames) > 0 {
		checkPaths = fnames
	}

	for _, path := range checkPaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}

		dir := absPath
		fi, err := os.Stat(dir)
		if err == nil && !fi.IsDir() {
			dir = filepath.Dir(dir)
		}

		gitDir := findGitDirRecursive(dir)
		if gitDir != "" {
			return gitDir, nil
		}
	}

	return "", fmt.Errorf("no git repository found")
}

func findGitDirRecursive(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func (gr *GitRepo) generateCommitMessage(diffs, context string) string {
	if gr.commitPrompt == "" {
		return "Update files" // Default message
	}

	// In a real implementation, you might want to use an LLM or other logic
	// to generate a meaningful commit message based on the diffs and context
	return fmt.Sprintf("Changes: %s", context)
}

func (gr *GitRepo) getCommitterName(aiderEdits bool) string {
	// Get the git config user.name
	cfg, err := gr.repo.Config()
	if err != nil {
		return "unknown"
	}

	userName := "unknown"
	if cfg.User.Name != "" {
		userName = cfg.User.Name
	}

	if aiderEdits && gr.attributeAuthor {
		return fmt.Sprintf("%s (aider)", userName)
	}
	return userName
}
