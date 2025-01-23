package coder

import (
	"io"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/coder/models"
	"github.com/y0ug/ai-helper/internal/coder/prompts"
	"github.com/y0ug/ai-helper/internal/coder/repomanager"
	"github.com/y0ug/ai-helper/internal/coder/settings"
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
