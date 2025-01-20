package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/y0ug/ai-helper/internal/llmagent"
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
	cm *llmagent.ConversationManager,
	requestCtx *llmcontext.RequestContext,
) error {
	// Extract code blocks from previous messages
	var searchReplace []string
	for _, msg := range cm.GetHistory() {
		if msg.Role == "assistant" {
			blocks := extractSearchReplaceBlocks(msg.Content.String())
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
	cm *llmagent.ConversationManager,
	response []*chat.ChatResponse,
) error {
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
