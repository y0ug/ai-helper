package extractor

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type EditBlockExtractor struct {
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

func NewEditBlockExtractor(
	logger *slog.Logger,
	format ExtractorType,
	fence Fence,
) *EditBlockExtractor {
	return &EditBlockExtractor{
		fence:  fence,
		logger: logger,
		format: format,
		name:   "EditBlockExtractor",
	}
}

func (c *EditBlockExtractor) GetName() string {
	return c.name
}

func (c *EditBlockExtractor) GetFormat() ExtractorType {
	return c.format
}

func (c *EditBlockExtractor) SetFence(fence Fence) {
	c.fence = fence
}

func (c *EditBlockExtractor) GetChatTools() []chat.Tool {
	return nil
}

func (c *EditBlockExtractor) Extract(
	msg *chat.ChatMessage,
) ([]actions.Action[any], error) {
	results := make([]actions.Action[any], 0)
	for _, content := range msg.Content {
		if content.Type == chat.ContentTypeText {
			actions := c.getEdits(content.String())
			results = append(results, actions...)
		}
	}
	return results, nil
}

func NewActionEdit(edit actions.ApplyEdit) actions.Action[any] {
	return actions.NewParsedAction(edit)
}

func (c *EditBlockExtractor) getEdits(content string) []actions.Action[any] {
	var results []actions.Action[any]
	lines := strings.Split(content, "\n")
	i := 0

	for i < len(lines) {
		line := lines[i]
		if isShellBlockStart(line) {
			cmd, newI := extractShellCommand(lines, i)
			action := actions.NewParsedAction(actions.ShellCommand{Command: cmd})
			results = append(results, action)
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
				results = append(results, NewActionEdit(*edit))
			}
			i = newI
			continue
		}
		i++
	}
	return results
}

func (c *EditBlockExtractor) extractEditBlock(
	lines []string,
	start int,
) (*actions.ApplyEdit, int, error) {
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

	edit := &actions.ApplyEdit{
		Filename: filename,
		Original: strings.Join(original, "\n"),
		Updated:  strings.Join(updated, "\n"),
	}
	return edit, i, nil
}

func (c *EditBlockExtractor) findFilename(lines []string, current int) string {
	// Look in previous 3 lines for filename
	for i := current - 1; i >= 0 && i >= current-3; i-- {
		fname := strings.TrimSpace(lines[i])
		if fname != "" && !strings.HasPrefix(fname, c.fence[0]) {
			return fname
		}
	}
	return ""
}
