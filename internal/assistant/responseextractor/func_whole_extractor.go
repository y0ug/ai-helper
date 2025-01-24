package responseextractor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type FuncWholeExtractor struct {
	logger *slog.Logger
	format ExtractorType
	name   string
	tools  map[string]Tooler
}

type WriteFileInput struct {
	Explanation string `json:"explanation" jsonschema_description:"Step by step plan for the changes to be made to the code (future tense, markdown format)"`
	Filename    string `json:"filename"    jsonschema_description:"Name of the file with the path to write to"`
	Content     string `json:"content"     jsonschema_description:"Content to write to the file"`
}

func NewFuncWholeExtractor(
	logger *slog.Logger,
	format ExtractorType,
	fence Fence,
) *FuncWholeExtractor {
	c := &FuncWholeExtractor{
		logger: logger,
		format: format,
		name:   "SingleWholeFileFuncService",
		tools:  make(map[string]Tooler),
	}
	tool := NewTool(
		"write_file",
		"Write content to a file",
		c.WriteFileHandler,
	)
	c.tools[tool.GetName()] = tool
	return c
}

func (c *FuncWholeExtractor) GetChatTools() []chat.Tool {
	tools := make([]chat.Tool, 0)
	for _, tool := range c.tools {
		tools = append(tools, tool.GetChatTool())
	}
	return tools
}

func (c *FuncWholeExtractor) WriteFileHandler(
	ctx context.Context,
	input WriteFileInput,
) (actions.Action[any], error) {
	c.logger.Debug("WriteFileHandler", "Explanation", input.Explanation)
	action := NewActionEdit(actions.ApplyEdit{
		Filename: input.Filename,
		Updated:  input.Content,
	})
	return action, nil
}

func (c *FuncWholeExtractor) GetName() string {
	return c.name
}

func (c *FuncWholeExtractor) GetFormat() ExtractorType {
	return c.format
}

func (c *FuncWholeExtractor) SetFence(fence Fence) {
}

func (c *FuncWholeExtractor) Extract(
	msg *chat.ChatMessage,
) ([]actions.Action[any], error) {
	results := make([]actions.Action[any], 0)
	// fmt.Println("FuncWholeExtractor", msg)
	for _, content := range msg.Content {
		if content.Type == chat.ContentTypeToolUse {
			action, err := c.processToolCall(content)
			if err != nil {
				c.logger.Error("Error processing tool call", "error", err)
				// toolsResultContentError, err := chat.NewToolResultContentInterface(
				// 	content.ID,
				// 	NewActionError(err),
				// )
				if err != nil {
					c.logger.Error("Error processing tool call error", "error", err)
				}
				// results = append(results, action)
				// *NewAction(ActionApplyEditToolResult, toolsResultContentError),
			} else {
				results = append(results, *action)
			}
			// results := c.GetEdits(content.String())
			// edits = append(edits, results...)
		}
	}
	return results, nil
}

func (tp *FuncWholeExtractor) processToolCall(
	content *chat.MessageContent,
) (*actions.Action[any], error) {
	ctx := context.TODO()
	if content.GetType() != string(chat.ContentTypeToolUse) {
		return nil, fmt.Errorf("invalid tool call: no tool call data")
	}

	logger := tp.logger.With("content", content.Name)
	tool, exists := tp.tools[content.Name]
	if !exists {
		return nil, fmt.Errorf("unknown tool: %s", content.Name)
	}

	logger.Debug("Tool call",
		"name", content.Name,
		"id", content.ID,
		"input", string(content.Input))

	response, err := tool.Execute(ctx, content.Input)
	if err != nil {
		logger.Error("Error executing tool",
			"error", err,
			"name", content.Name)
		return nil, fmt.Errorf("error executing tool: %w", err)
	}

	var action actions.Action[actions.ApplyEdit]
	json.Unmarshal(response, &action)

	toolResultAction := actions.NewParsedAction(actions.ToolResultAction{
		ToolResult: *chat.NewToolResultContent(content.ID, ""),
		NextAction: actions.ToGeneric(action),
	})
	// tp.logger.Debug("Tool result", "json", string(response))
	// var results ParsedAction
	// logger.Debug("Tool result",
	// 	"name", content.Name)
	return &toolResultAction, nil
}
