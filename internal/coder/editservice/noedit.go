package editservice

import (
	"log/slog"

	"github.com/y0ug/ai-helper/internal/filemanager"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type NoEditService struct {
	logger *slog.Logger
	name   string
	format EditFormat
}

func NewNoEditService(logger *slog.Logger, format EditFormat) *NoEditService {
	return &NoEditService{name: "NoEditService", format: format}
}

func (c *NoEditService) GetName() string {
	return c.name
}

func (c *NoEditService) GetFormat() EditFormat {
	return c.format
}

func (c *NoEditService) GetChatTools() []chat.Tool {
	return nil
}

func (c *NoEditService) GetEditsMsg(msg *chat.ChatMessage) ([]EditResult, []chat.MessageContent) {
	edits := make([]EditResult, 0)
	for _, content := range msg.Content {
		if content.Type == chat.ContentTypeText {
			results := c.GetEdits(content.String())
			edits = append(edits, results...)
		}
	}
	return edits, nil
}

func (*NoEditService) GetEdits(content string) []EditResult {
	return nil
}

func (*NoEditService) ApplyEdits(filemanager.FileManager, []Edit, bool) error {
	return nil
}

func (*NoEditService) SetFence(Fence) {
}
