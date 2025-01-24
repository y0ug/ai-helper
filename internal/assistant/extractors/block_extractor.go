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

func NewShellExecActionWithConfirm(command string) []actions.Action[any] {
	msg := fmt.Sprintf(
		"Are you sure you want to run the following command?\n\n```\n%s\n```",
		command,
	)
	return []actions.Action[any]{
		actions.NewParsedAction(
			actions.AwaitUserInput{Question: msg, InputType: actions.UserInputTypeConfirm},
		),
		actions.NewParsedAction(actions.ShellCommand{Command: command}),
	}
}

func (c *BlockExtractor) getEdits(content string) []actions.Action[any] {
	var results []actions.Action[any]
	lines := strings.Split(content, "\n")
	i := 0

	for i < len(lines) {
		line := lines[i]
		if isShellBlockStart(line) {
			cmd, newI := extractShellCommand(lines, i)
			action := NewShellExecActionWithConfirm(cmd)
			results = append(results, action...)
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
				results = append(results, NewActionApplyEdit(*edit))
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
