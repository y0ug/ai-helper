package editblock

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (c *EditBlockCoder) FindEditBlocks(content string) []EditResult {
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

func (c *EditBlockCoder) extractEditBlock(
	lines []string,
	start int,
) (string, string, string, int, error) {
	// Find filename in preceding lines
	filename := c.findFilename(lines, start)

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

func (c *EditBlockCoder) findFilename(lines []string, current int) string {
	// Look in previous 3 lines for filename
	for i := current - 1; i >= 0 && i >= current-3; i-- {
		fname := strings.TrimSpace(lines[i])
		if fname != "" && !strings.HasPrefix(fname, c.Fence[0]) {
			return fname
		}
	}
	return "unknown"
}

func (c *EditBlockCoder) ApplyEdits(edits []EditBlock) error {
	for _, edit := range edits {
		fullPath := filepath.Join(c.RootPath, edit.Filename)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return err
		}

		newContent := DoReplace(string(content), edit.Original, edit.Updated)
		if newContent != string(content) {
			err = os.WriteFile(fullPath, []byte(newContent), 0644)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
