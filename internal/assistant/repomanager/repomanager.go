package repomanager

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/extractors"
	"github.com/y0ug/ai-helper/internal/filemanager"
	"github.com/y0ug/ai-helper/pkg/gitrepo"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

// Fence represents a pair of opening and closing delimiters for code blocks

type RepoManager struct {
	fence            extractors.Fence
	root             string
	logger           *slog.Logger
	fm               filemanager.FileManager
	git              gitrepo.GitRepoInterface
	absRootPathCache map[string]string
}

var _ RepoManagerInterface = &RepoManager{}

type RepoManagerInterface interface {
	GetFM() filemanager.FileManager
	GetGit() gitrepo.GitRepoInterface
	GetFence() [2]string
	GetRoot() string
	ChooseFence()
	GetFilesContent() string
	GetReadOnlyFilesContent() string
	GetRepoMap() string
	Process(*chat.ChatMessage) (ProcessType, []chat.MessageContent, error)
	ApplyEdit(edits actions.ApplyEdit) error
}
type ProcessType string

var (
	ProcessTypeEdit       ProcessType = "edit"
	ProcessTypeFuncResult ProcessType = "func-result"
	ProcessTypeNone       ProcessType = "none"
)

func NewRepoManager(
	root string,
	logger *slog.Logger,
	fm filemanager.FileManager,
	git gitrepo.GitRepoInterface,
) *RepoManager {
	return &RepoManager{
		root:             root,
		logger:           logger,
		fm:               fm,
		git:              git,
		absRootPathCache: make(map[string]string),
	}
}

func (c *RepoManager) GetRoot() string {
	return c.root
}

func (c *RepoManager) Process(msg *chat.ChatMessage) (ProcessType, []chat.MessageContent, error) {
	// if c.editSvc == nil {
	return ProcessTypeNone, nil, nil
	// }

	// // TODO: set it here or when we ChooseFence
	// c.editSvc.SetFence(c.fence)
	//
	// results, msgs := c.editSvc.Extract(msg)
	// // if len(msgs) > 0 {
	// // 	return ProcessTypeEdit, msgs, nil
	// // }
	// // if len(results) == 0 {
	// // 	return ProcessTypeNone, nil, nil
	// // }
	// var edits []responseextractor.Edit
	// for _, result := range results {
	// 	if result.Edit != nil {
	// 		edits = append(edits, *result.Edit)
	// 	}
	// }
	//
	// // err := c.editSvc.ApplyEdits(c.fm, edits, true)
	// // if err != nil {
	// // 	return ProcessTypeNone, nil, fmt.Errorf("failed to apply edits dry run %w", err)
	// // }
	// err := c.editSvc.ApplyEdits(c.fm, edits, false)
	// if err != nil {
	// 	return ProcessTypeNone, nil, fmt.Errorf("failed to apply edits %w", err)
	// }
	//
	// return ProcessTypeNone, msgs, nil
}

func (c *RepoManager) GetRepoMap() string {
	return ""
}

func (c *RepoManager) GetGit() gitrepo.GitRepoInterface {
	return c.git
}

func (c *RepoManager) GetFM() filemanager.FileManager {
	return c.fm
}

func (c *RepoManager) GetFence() [2]string {
	return c.fence
}

// chooseFence selects appropriate fence markers that won't conflict with file contents
func (c *RepoManager) ChooseFence() {
	// Get all content from files to check for fence conflicts
	allContent := c.getAllContent()

	// Try each fence option until we find one that doesn't appear in the content
	for _, fence := range extractors.DefaultFences {
		if !hasFenceConflict(allContent, fence) {
			c.fence = fence
			return
		}
	}

	// If all fences conflict (unlikely), use the default and warn
	c.fence = extractors.DefaultFences[0]
	c.logger.Warn(
		"Unable to find a non-conflicting fence strategy! Falling back", "fence", c.fence)
}

// getAllContent combines content from all files being handled
func (c *RepoManager) getAllContent() string {
	var builder strings.Builder

	// Get all files
	files := c.fm.List(filemanager.NewFileFilters(filemanager.FilterAll))

	for _, info := range files {
		builder.WriteString(info.Content)
		builder.WriteString("\n")
	}

	return builder.String()
}

// hasFenceConflict checks if fence markers appear in the content
func hasFenceConflict(content string, fence extractors.Fence) bool {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, fence[0]) || strings.HasPrefix(line, fence[1]) {
			return true
		}
	}
	return false
}

// getRelativePath converts an absolute path to a path relative to the repository root or working directory
func (c *RepoManager) getRelativePath(absPath string) string {
	// Check cache first
	if relPath, ok := c.absRootPathCache[absPath]; ok {
		return relPath
	}

	// Get relative path
	relPath, err := filepath.Rel(c.root, absPath)
	if err != nil {
		// If we can't get relative path, return absolute path
		return absPath
	}

	// Cache and return the result
	c.absRootPathCache[absPath] = relPath
	return relPath
}

// absRootPath converts a relative path to absolute path using root directory
func (c *RepoManager) absRootPath(path string) string {
	// Check cache first
	if cached, ok := c.absRootPathCache[path]; ok {
		return cached
	}

	// Join with root and get absolute path
	absPath := filepath.Join(c.root, path)
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		// If we can't get absolute path, return joined path
		absPath = filepath.Join(c.root, path)
	}

	// Cache and return result
	c.absRootPathCache[path] = absPath
	return absPath
}

func (c *RepoManager) GetFilesContent() string {
	var content string
	files := c.fm.List(filemanager.NewFileFilters(filemanager.FilterEditable))

	for fname, info := range files {
		relPath := c.getRelativePath(fname)
		content += "\n" + relPath + "\n"
		content += c.fence[0] + "\n"
		content += info.Content
		content += c.fence[1] + "\n"
	}
	return content
}

func (c *RepoManager) GetReadOnlyFilesContent() string {
	var content string
	files := c.fm.List(filemanager.NewFileFilters(filemanager.FilterReadOnly))

	for fname, info := range files {
		relPath := c.getRelativePath(fname)
		content += "\n" + relPath + "\n"
		content += c.fence[0] + "\n"
		content += info.Content
		content += c.fence[1] + "\n"
	}
	return content
}

func (c *RepoManager) ApplyEdit(edit actions.ApplyEdit) error {
	content, isEditable, err := c.fm.Get(edit.Filename)
	if err != nil {
		c.logger.Error("error reading file ", "filename", edit.Filename, "error", err)
		c.logger.Info("file is not in the file list adding it", "filename", edit.Filename)
		content = ""
		c.fm.Add(edit.Filename, false)
	} else if !isEditable {
		c.logger.Info("file is read-only we will not edit it", "filename", edit.Filename)
		return fmt.Errorf("file %s is read-only we will not edit it", edit.Filename)
	}

	newContent := extractors.ApplyEdit(string(content), edit.Original, edit.Updated)
	if newContent == string(content) {
		err = fmt.Errorf("new content is the same as the original content")
	}

	c.logger.Info(
		"applying edit to file",
		"filename",
		edit.Filename,
		"content",
		string(content),
		"new_content",
		newContent,
		"original",
		edit.Original,
		"updated",
		edit.Updated,
	)
	if newContent != string(content) {
		err := c.fm.Write(edit.Filename, newContent)
		if err != nil {
			return fmt.Errorf("error writing file %s: %w", edit.Filename, err)
		}
	}

	return nil
}
