package assistant

import (
	"io"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/assistant/models"
	"github.com/y0ug/ai-helper/internal/assistant/prompts"
	"github.com/y0ug/ai-helper/internal/assistant/repomanager"
	"github.com/y0ug/ai-helper/internal/assistant/settings"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type CoderOptions struct {
	MainModel    *models.Model
	RepoManager  repomanager.RepoManagerInterface
	LlmClient    chat.Provider
	Logger       *slog.Logger
	Prompts      prompts.Prompter
	Settings     *settings.CoderSettings
	StreamWriter io.Writer
	Stream       bool
}
