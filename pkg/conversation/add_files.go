package conversation

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type AddFileHandler struct{}

func NewAddFileHandler() *AddFileHandler {
	return &AddFileHandler{}
}

func (h *AddFileHandler) PreProcess(ctx context.Context, cm Manager) error {
	// Read the user-provided "Input" variable, which should have filenames
	inputVal, _ := cm.GetCtx().GetVariable("Input")
	filenames := parseFilenames(inputVal)

	// Add the files
	fm := cm.GetCtx().GetFM()
	for _, fn := range filenames {
		if err := fm.Add(fn, false); err != nil {
			fmt.Printf("Error loading file %s: %v\n", fn, err)
		} else {
			fmt.Printf("Added file: %s\n", fn)
		}
	}
	return nil
}

func (h *AddFileHandler) PostProcess(
	ctx context.Context,
	cm Manager,
	responses []*chat.ChatResponse,
	w io.Writer,
) error {
	// Possibly do something after the LLM responds
	return nil
}

func parseFilenames(input interface{}) []string {
	// Convert to string, then parse space or CSV...
	if s, ok := input.(string); ok {
		return strings.Fields(s)
	}
	return nil
}
