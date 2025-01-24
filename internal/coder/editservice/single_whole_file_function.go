package editservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/filemanager"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type SingleWholeFileFuncService struct {
	logger *slog.Logger
	format EditFormat
	name   string
	tools  map[string]Tooler
}

// func NewWriteFile() *ToolBase[WriteFileInput] {
// 	inputSchema := GenerateSchema[WriteFileInput]()
// 	return &ToolBase[WriteFileInput]{
// 		Name:        "write_file",
// 		InputSchema: inputSchema,
// 		Description: "Write content to a file",
// 		handler:     WriteFileHandler,
// 	}
// }

type WriteFileInput struct {
	Explanation string `json:"explanation" jsonschema_description:"Step by step plan for the changes to be made to the code (future tense, markdown format)"`
	Filename    string `json:"filename"    jsonschema_description:"Name of the file with the path to write to"`
	Content     string `json:"content"     jsonschema_description:"Content to write to the file"`
}

func NewSingleWholeFileFuncService(
	logger *slog.Logger,
	format EditFormat,
	fence Fence,
) *SingleWholeFileFuncService {
	c := &SingleWholeFileFuncService{
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

func (c *SingleWholeFileFuncService) GetChatTools() []chat.Tool {
	tools := make([]chat.Tool, 0)
	for _, tool := range c.tools {
		tools = append(tools, tool.GetChatTool())
	}
	return tools
}

type WriteFileResult struct {
	Msg  string `json:"msg"`
	Edit Edit   `json:"edit"`
}

func (c *SingleWholeFileFuncService) WriteFileHandler(
	ctx context.Context,
	input WriteFileInput,
) (WriteFile, error) {
	result := fmt.Sprintf("Wrote file %s and content length %d",
		input.Filename,
		input.Explanation,
		len(input.Content))
	edit := Edit{
		Filename: input.Filename,
		Updated:  input.Content,
		Format:   c.format,
	}
	c.logger.Info("WriteFileHandler", "input", input)
	return WriteFileResult{Msg: result, Edit: edit}, nil
}

func (c *SingleWholeFileFuncService) GetName() string {
	return c.name
}

func (c *SingleWholeFileFuncService) GetFormat() EditFormat {
	return c.format
}

func (c *SingleWholeFileFuncService) SetFence(fence Fence) {
}

func (c *SingleWholeFileFuncService) GetEditsMsg(
	msg *chat.ChatMessage,
) ([]EditResult, []chat.MessageContent) {
	edits := make([]EditResult, 0)
	msgs := make([]chat.MessageContent, 0)
	for _, content := range msg.Content {
		if content.Type == chat.ContentTypeToolUse {
			m, err := c.processToolCall(content)
			if err != nil {
				c.logger.Error("Error processing tool call", "error", err)
			}
			msgs = append(msgs, *m)
			// results := c.GetEdits(content.String())
			// edits = append(edits, results...)
		}
	}
	return edits, msgs
}

func (c *SingleWholeFileFuncService) GetEdits(content string) []EditResult {
	return nil
}

func (tp *SingleWholeFileFuncService) processToolCall(
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

	b, err := json.Marshal(response)
	if err != nil {
		logger.Error("Failed to Marshall response",
			"error", err,
			"name", content.Name)
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	logger.Debug("Tool result",
		"name", content.Name,
		"result", response)
	return chat.NewToolResultContent(content.ID, string(b)), nil
}

func (c *SingleWholeFileFuncService) ApplyEdits(
	fm filemanager.FileManager,
	edits []Edit,
	dryrun bool,
) error {
	for _, edit := range edits {
		_, isEditable, err := fm.Get(edit.Filename)
		if err != nil {
			c.logger.Info("Adding new file", "filename", edit.Filename)
			fm.Add(edit.Filename, false)
		} else if !isEditable {
			c.logger.Info("File is read-only", "filename", edit.Filename)
			continue
		}

		if !dryrun {
			if err := fm.Write(edit.Filename, edit.Updated); err != nil {
				return fmt.Errorf("failed to write file %s: %w", edit.Filename, err)
			}
			c.logger.Info("Applied edit", "filename", edit.Filename)
		}
	}
	return nil
}
