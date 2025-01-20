package llmagent

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/y0ug/ai-helper/internal/coder/diff"
	"github.com/y0ug/ai-helper/internal/coder/parser"
	"github.com/y0ug/ai-helper/internal/llmcontext"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
)

// CodeDiffHandler processes code diffs in the conversation
type CodeDiffHandler struct{}

// NewCodeDiffHandler creates a new CodeDiffHandler
func NewCodeDiffHandler() *CodeDiffHandler {
	return &CodeDiffHandler{}
}

// PreProcess handles pre-processing of code diffs
func (h *CodeDiffHandler) PreProcess(
	ctx context.Context,
	cm *ConversationManager,
	requestCtx *llmcontext.RequestContext,
) error {
	// Extract code blocks from previous messages
	var searchReplace []string
	for _, msg := range cm.GetHistory() {
		if msg.Role == "assistant" {
			blocks := extractSearchReplaceBlocks(msg.Content[0].String())
			searchReplace = append(searchReplace, blocks...)
		}
	}

	if len(searchReplace) > 0 {
		requestCtx.Vars["CodeDiffs"] = strings.Join(searchReplace, "\n")
	}
	return nil
}

// PostProcess handles post-processing of code diffs
func (h *CodeDiffHandler) PostProcess(
	ctx context.Context,
	cm *ConversationManager,
	response []*chat.ChatResponse,
	w io.Writer,
) error {
	fmt.Println("CodeDiffHandler PostProcess")
	diff := diff.NewGenerator()
	parser := parser.New()
	if len(response) > 0 && len(response[0].Choice) > 0 && len(response[0].Choice[0].Content) > 0 {
		sections := parser.ParseResponse(response[0].Choice[0].Content[0].String())
		modifiedFiles, err := diff.ApplyChanges(cm.Files, sections)
		if err != nil {
			fmt.Println("Error applying changes:", err)
			return nil
		}
		for filename, file := range modifiedFiles {
			// fmt.Println("filename:", filename)
			// fmt.Println("file:", file)
			if orginalContent, exists := cm.Files[filename]; exists && orginalContent != file {
				// patches[filename] = diff.GeneratePatch(orginalContent, file)
				ext := filepath.Ext(filename)[1:]
				fmt.Fprintf(w, "```%s\n%s\n```\n", ext, file)
			}
		}
	} else {
		fmt.Println("response is empty")
	}

	return nil
}

// extractSearchReplaceBlocks extracts code blocks between SEARCH and REPLACE markers
func extractSearchReplaceBlocks(content string) []string {
	var blocks []string
	lines := strings.Split(content, "\n")
	inBlock := false
	var currentBlock strings.Builder

	for _, line := range lines {
		if strings.Contains(line, "<<<<<<< SEARCH") {
			inBlock = true
			currentBlock.Reset()
		}
		if inBlock {
			currentBlock.WriteString(line + "\n")
		}
		if strings.Contains(line, ">>>>>>> REPLACE") {
			inBlock = false
			blocks = append(blocks, currentBlock.String())
		}
	}
	return blocks
}
