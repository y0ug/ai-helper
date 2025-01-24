package responseextractor

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type BlockExtractor struct {
	fence  Fence
	logger *slog.Logger
	format ExtractorType
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

func NewBlockExtractor(
	logger *slog.Logger,
	format ExtractorType,
	fence Fence,
) *BlockExtractor {
	return &BlockExtractor{
		fence:  fence,
		logger: logger,
		format: format,
		name:   "EditBlockExtractor",
	}
}

func (c *BlockExtractor) GetName() string {
	return c.name
}

func (c *BlockExtractor) GetFormat() ExtractorType {
	return c.format
}

func (c *BlockExtractor) SetFence(fence Fence) {
	c.fence = fence
}

func (c *BlockExtractor) GetChatTools() []chat.Tool {
	return nil
}

func NewActionEdit(edit *Edit) *ParsedAction {
	return NewAction(ActionApplyEdit, edit)
}

func NewAction(actionType ActionType, input interface{}) *ParsedAction {
	inputPayload, err := json.Marshal(input)
	if err != nil {
		return nil
	}
	return &ParsedAction{
		Type:  actionType,
		Input: inputPayload,
	}
}

func NewActionShellCommand(cmd string) *ParsedAction {
	return NewAction(ActionShellCmd, &ShellCommand{
		Command: cmd,
	})
}

func NewActionError(err error) *ParsedAction {
	return NewAction(ActionApplyError, err)
}

func (c *BlockExtractor) Extract(
	msg *chat.ChatMessage,
) ([]ParsedAction, error) {
	results := make([]ParsedAction, 0)
	for _, content := range msg.Content {
		if content.Type == chat.ContentTypeText {
			actions := c.getEdits(content.String())
			results = append(results, actions...)
		}
	}
	return results, nil
}

func (c *BlockExtractor) getEdits(content string) []ParsedAction {
	var results []ParsedAction
	lines := strings.Split(content, "\n")
	i := 0

	for i < len(lines) {
		line := lines[i]
		if isShellBlockStart(line) {
			cmd, newI := extractShellCommand(lines, i)
			results = append(results, *NewActionShellCommand(cmd))
			i = newI
			continue
		}

		if headRe.MatchString(strings.TrimSpace(line)) {
			edit, newI, err := c.extractEditBlock(lines, i)
			if err != nil {
				results = append(
					results,
					*NewActionError(NewExtractorError(c.GetName(), c.GetFormat(), err, newI)),
				)
			} else {
				results = append(results, *NewActionEdit(edit))
			}
			i = newI
			continue
		}
		i++
	}
	return results
}

// TODO: should return an EditResult
func (c *BlockExtractor) extractEditBlock(
	lines []string,
	start int,
) (*Edit, int, error) {
	// Find filename in preceding lines
	filename := c.findFilename(lines, start)
	if filename == "" {
		return nil, start, fmt.Errorf("filename not found")
	}

	// Extract original and updated blocks
	var original, updated []string
	i := start + 1
	for ; i < len(lines) && !dividerRe.MatchString(strings.TrimSpace(lines[i])); i++ {
		original = append(original, lines[i])
	}

	if i >= len(lines) {
		return nil, i, fmt.Errorf("missing divider")
	}
	i++ // Skip divider

	for ; i < len(lines) && !replaceRe.MatchString(strings.TrimSpace(lines[i])); i++ {
		updated = append(updated, lines[i])
	}

	if i >= len(lines) {
		return nil, i, fmt.Errorf("missing replace marker")
	}
	i++ // Skip replace marker

	edit := &Edit{
		Filename: filename,
		Original: strings.Join(original, "\n"),
		Updated:  strings.Join(updated, "\n"),
	}
	return edit, i, nil
}

func (c *BlockExtractor) findFilename(lines []string, current int) string {
	// Look in previous 3 lines for filename
	for i := current - 1; i >= 0 && i >= current-3; i-- {
		fname := strings.TrimSpace(lines[i])
		if fname != "" && !strings.HasPrefix(fname, c.fence[0]) {
			return fname
		}
	}
	return ""
}

// func (c *EditBlockExtractor) ApplyEdits(
// 	fm filemanager.FileManager,
// 	edits []Edit,
// 	dryrun bool,
// ) error {
// 	for _, edit := range edits {
// 		content, isEditable, err := fm.Get(edit.Filename)
// 		if err != nil {
// 			c.logger.Error("error reading file ", "filename", edit.Filename, "error", err)
// 			c.logger.Info("file is not in the file list adding it", "filename", edit.Filename)
// 			content = ""
// 			fm.Add(edit.Filename, false)
// 		} else if !isEditable {
// 			c.logger.Info("file is read-only we will not edit it", "filename", edit.Filename)
// 			continue
// 		}
//
// 		newContent := DoReplace(string(content), edit.Original, edit.Updated)
// 		c.logger.Info(
// 			"applying edit to file",
// 			"filename",
// 			edit.Filename,
// 			"content",
// 			string(content),
// 			"new_content",
// 			newContent,
// 			"original",
// 			edit.Original,
// 			"updated",
// 			edit.Updated,
// 		)
// 		if newContent != string(content) {
// 			if !dryrun {
// 				err := fm.Write(edit.Filename, newContent)
// 				if err != nil {
// 					return err
// 				}
// 			}
// 		}
// 	}
// 	return nil
// }
