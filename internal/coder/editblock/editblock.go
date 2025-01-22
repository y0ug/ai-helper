package editblock

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/transport/file"
	"github.com/y0ug/ai-helper/internal/coder/repomanager"
	"github.com/y0ug/ai-helper/internal/filemanager"
)

type EditBlockService struct {
	fence  repomanager.Fence
	logger *slog.Logger
	fm     filemanager.FileManager
}

type EditBlock struct {
	Filename string
	Original string
	Updated  string
}

type ShellCommand struct {
	Command string
}

type EditResult struct {
	Edit  *EditBlock
	Shell *ShellCommand
	Err   error
}

const (
	searchMarker  = "<<<<<<< SEARCH"
	divider       = "======="
	replaceMarker = ">>>>>>> REPLACE"
)

var (
	headRe    = regexp.MustCompile(`^<{5,9} SEARCH\s*$`)
	dividerRe = regexp.MustCompile(`^={5,9}\s*$`)
	replaceRe = regexp.MustCompile(`^>{5,9} REPLACE\s*$`)
)

func NewBlockService(logger *slog.Logger, fence repomanager.Fence) *EditBlockService {
	return &EditBlockService{
		fence:  fence 
		logger: logger,
	}
}


func (c *EditBlockService) GetEdits(content string) []EditResult {
	var results []EditResult
	lines := strings.Split(content, "\n")
	i := 0

	for i < len(lines) {
		line := lines[i]
		if isShellBlockStart(line) {
			cmd, newI := extractShellCommand(lines, i)
			results = append(results, EditResult{Shell: &ShellCommand{Command: cmd}})
			i = newI
			continue
		}

		if headRe.MatchString(strings.TrimSpace(line)) {
			filename, original, updated, newI, err := c.extractEditBlock(lines, i)
			if err != nil {
				results = append(results, EditResult{Err: err})
			} else {
				results = append(results, EditResult{
					Edit: &EditBlock{
						Filename: filename,
						Original: original,
						Updated:  updated,
					},
				})
			}
			i = newI
			continue
		}
		i++
	}
	return results
}

// TODO: should return an EditResult
func (c *EditBlockService) extractEditBlock(
	lines []string,
	start int,
) (string, string, string, int, error) {
	// Find filename in preceding lines
	filename := c.findFilename(lines, start)
  if filename == "" {
		return "", "", "", i, fmt.Errorf("filename not found")
  }

	// Extract original and updated blocks
	var original, updated []string
	i := start + 1
	for ; i < len(lines) && !dividerRe.MatchString(strings.TrimSpace(lines[i])); i++ {
		original = append(original, lines[i])
	}

	if i >= len(lines) {
		return "", "", "", i, fmt.Errorf("missing divider")
	}
	i++ // Skip divider

	for ; i < len(lines) && !replaceRe.MatchString(strings.TrimSpace(lines[i])); i++ {
		updated = append(updated, lines[i])
	}

	if i >= len(lines) {
		return "", "", "", i, fmt.Errorf("missing replace marker")
	}
	i++ // Skip replace marker

	return filename, strings.Join(original, "\n"), strings.Join(updated, "\n"), i, nil
}

func (c *EditBlockService) findFilename(lines []string, current int) string {
	// Look in previous 3 lines for filename
	for i := current - 1; i >= 0 && i >= current-3; i-- {
		fname := strings.TrimSpace(lines[i])
		if fname != "" && !strings.HasPrefix(fname, c.fence[0]) {
			return fname
		}
	}
	return ""
}

func (c *EditBlockService) ApplyEdits(edits []EditBlock, dryrun bool, fm filemanager.FileManager) error {
	for _, edit := range edits {
    content, isReadOnly, err := fm.Get(edit.Filename)
    if err != nil {
      c.logger.Errorf("error reading file %s: %v", edit.Filename, err)
      c.logger.Info("file %s is not in the file list")
    }
    if isReadOnly {
      c.logger.Infof("file %s is read-only we will not edit it", edit.Filename)
      continue
    }
		fullPath := filepath.Join(c.RootPath, edit.Filename)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return err
		}

		newContent := DoReplace(string(content), edit.Original, edit.Updated)
		if newContent != string(content) {
			if !dryrun {
				err = os.WriteFile(fullPath, []byte(newContent), 0644)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
