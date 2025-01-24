package extractors

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
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
	}
}

func (c *BlockExtractor) Name() string {
	return "BlockExtractor"
}

func (c *BlockExtractor) SupportedActions() []actions.ActionType {
	return []actions.ActionType{"apply_edit", "shell_command"}
}

func (c *BlockExtractor) Type() ExtractorType {
	return c.format
}

func (c *BlockExtractor) GetChatTools() []chat.Tool {
	return nil
}

func (c *BlockExtractor) SetFence(fence Fence) {
	c.fence = fence
}

func (c *BlockExtractor) Extract(
	msg *chat.ChatMessage,
) ([]actions.Action, error) {
	results := make([]actions.Action, 0)
	for _, content := range msg.Content {
		if content.Type == chat.ContentTypeText {
			actions := c.getEdits(content.String())
			results = append(results, actions...)
		}
	}
	return results, nil
}

func (c *BlockExtractor) getEdits(content string) []actions.Action {
	var results []actions.Action
	lines := strings.Split(content, "\n")
	i := 0

	for i < len(lines) {
		line := lines[i]
		if isShellBlockStart(line) {
			cmd, newI := extractShellCommand(lines, i)
			results = append(results, actions.NewShellCommand(nil, cmd, false))
			i = newI
			continue
		}

		if headRe.MatchString(strings.TrimSpace(line)) {
			edit, newI, err := c.extractEditBlock(lines, i)
			if err != nil {
				// action := actions.NewParsedAction(actions.Error{Command: cmd})
				// results = append(
				// 	results,
				// 	*NewActionError(NewExtractorError(c.GetName(), c.GetFormat(), err, newI)),
				// )
			} else {
				results = append(results, edit)
			}
			i = newI
			continue
		}
		i++
	}
	return results
}

func (c *BlockExtractor) extractEditBlock(
	lines []string,
	start int,
) (actions.Action, int, error) {
	// Find filename in preceding lines
	filename := c.findFilename(lines, start)
	if filename == "" {
		return actions.Action{}, start, fmt.Errorf("filename not found")
	}

	// Extract original and updated blocks
	var original, updated []string
	i := start + 1
	for ; i < len(lines) && !dividerRe.MatchString(strings.TrimSpace(lines[i])); i++ {
		original = append(original, lines[i])
	}

	if i >= len(lines) {
		return actions.Action{}, i, fmt.Errorf("missing divider")
	}
	i++ // Skip divider

	for ; i < len(lines) && !replaceRe.MatchString(strings.TrimSpace(lines[i])); i++ {
		updated = append(updated, lines[i])
	}

	if i >= len(lines) {
		return actions.Action{}, i, fmt.Errorf("missing replace marker")
	}
	i++ // Skip replace marker

	originalStr := strings.Join(original, "\n")
	updatedStr := strings.Join(updated, "\n")
	return actions.NewApplyEdit(nil, filename, originalStr, updatedStr), i, nil
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
