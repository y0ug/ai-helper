package coder

import (
	"log/slog"

	"github.com/y0ug/ai-helper/internal/coder/models"
	"github.com/y0ug/ai-helper/internal/coder/prompts"
	"github.com/y0ug/ai-helper/internal/coder/repomanager"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
)

type CoderOptions struct {
	MainModel   *models.Model
	RepoManager repomanager.RepoManagerInterface
	LlmClient   chat.Provider
	Logger      *slog.Logger
	Prompts     prompts.Prompter
}
