package editservice

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/y0ug/ai-helper/internal/filemanager"
)

type EditBlockService struct {
	fence  Fence
	logger *slog.Logger
	format EditFormat
	name   string
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

func NewEditBlockService(logger *slog.Logger, format EditFormat, fence Fence) *EditBlockService {
	return &EditBlockService{
		fence:  fence,
		logger: logger,
		format: format,
		name:   "EditBlockService",
	}
}

func (c *EditBlockService) GetName() string {
	return c.name
}

func (c *EditBlockService) GetFormat() EditFormat {
	return c.format
}

func (c *EditBlockService) SetFence(fence Fence) {
	c.fence = fence
}

func (c *EditBlockService) newEdit(filename, original, updated string) *Edit {
	return &Edit{
		Filename: filename,
		Original: original,
		Updated:  updated,
		Format:   c.format,
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
					Edit: c.newEdit(filename, original, updated),
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
		return "", "", "", start, fmt.Errorf("filename not found")
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

func (c *EditBlockService) ApplyEdits(
	fm filemanager.FileManager,
	edits []Edit,
	dryrun bool,
) error {
	for _, edit := range edits {
		content, isEditable, err := fm.Get(edit.Filename)
		if err != nil {
			c.logger.Error("error reading file ", "filename", edit.Filename, "error", err)
			c.logger.Info("file is not in the file list adding it", "filename", edit.Filename)
			content = ""
			fm.Add(edit.Filename, false)
		} else if !isEditable {
			c.logger.Info("file is read-only we will not edit it", "filename", edit.Filename)
			continue
		}

		newContent := DoReplace(string(content), edit.Original, edit.Updated)
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
			if !dryrun {
				err := fm.Write(edit.Filename, newContent)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
