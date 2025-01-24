package responseextractor

import (
	"context"
	"fmt"
	"log/slog"

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
) (ParsedAction, error) {
	// c.logger.Debug("WriteFileHandler", "input", input)
	edit := &Edit{
		Filename: input.Filename,
		Updated:  input.Content,
	}
	return *NewActionEdit(edit), nil
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
) ([]ParsedAction, error) {
	results := make([]ParsedAction, 0)
	// fmt.Println("FuncWholeExtractor", msg)
	for _, content := range msg.Content {
		if content.Type == chat.ContentTypeToolUse {
			toolsResultContent, err := c.processToolCall(content)
			if err != nil {
				c.logger.Error("Error processing tool call", "error", err)
				toolsResultContentError, err := chat.NewToolResultContentInterface(
					content.ID,
					NewActionError(err),
				)
				if err != nil {
					c.logger.Error("Error processing tool call error", "error", err)
				}
				results = append(
					results,
					*NewAction(ActionApplyEditToolResult, toolsResultContentError),
				)
			} else {
				results = append(results, *NewAction(ActionApplyEditToolResult, toolsResultContent))
			}
			// results := c.GetEdits(content.String())
			// edits = append(edits, results...)
		}
	}
	return results, nil
}

func (tp *FuncWholeExtractor) processToolCall(
	content *chat.MessageContent,
) (*chat.MessageContent, error) {
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

	// tp.logger.Debug("Tool result", "json", string(response))
	// var results ParsedAction
	// json.Unmarshal(response, &results)
	// logger.Debug("Tool result",
	// 	"name", content.Name)
	return chat.NewToolResultContent(content.ID, string(response)), nil
}
