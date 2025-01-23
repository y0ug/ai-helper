package editservice

import (
	"log/slog"

	"github.com/y0ug/ai-helper/internal/filemanager"
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

func (*NoEditService) GetEdits(content string) []EditResult {
	return nil
}

func (*NoEditService) ApplyEdits(filemanager.FileManager, []Edit, bool) error {
	return nil
}

func (*NoEditService) SetFence(Fence) {
}
