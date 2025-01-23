package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/lmittmann/tint"
	"github.com/y0ug/ai-helper/internal/coder"
	modelinfocoder "github.com/y0ug/ai-helper/internal/coder/models"
	"github.com/y0ug/ai-helper/internal/coder/prompts"
	"github.com/y0ug/ai-helper/internal/coder/repomanager"
	"github.com/y0ug/ai-helper/internal/coder/settings"
	"github.com/y0ug/ai-helper/internal/consolecoder"
	"github.com/y0ug/ai-helper/internal/filemanager"
	"github.com/y0ug/ai-helper/pkg/gitrepo"
	"github.com/y0ug/ai-helper/pkg/highlighter"
	"github.com/y0ug/ai-helper/pkg/llmclient"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
	"github.com/y0ug/ai-helper/pkg/llmclient/modelinfo"
)

func main() {
	verbose := flag.Bool("v", false, "Show verbose output")
	flag.Parse()

	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}

	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      level,
		TimeFormat: time.Kitchen,
	}))

	// Setup model info provider
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "ai-helper")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		logger.Error("Error creating config directory", "error", err)
		os.Exit(1)
	}

	infoProviderCacheFile := filepath.Join(configDir, "provider_cache.json")
	infoProviders, err := modelinfo.New(infoProviderCacheFile)
	if err != nil {
		logger.Error("Error creating model info providers", "error", err)
		os.Exit(1)
	}
	model := os.Getenv("AI_MODEL")
	if model == "" {
		logger.Error("AI_MODEL environment variable not set")
		os.Exit(1)
	}

	modelInfo, err := modelinfo.Parse(model, infoProviders)
	if err != nil {
		logger.Error("Error parsing model info", "error", err)
		os.Exit(1)
	}
	// modelInfoCoderRegistry := modelinfocoder.InitializeDefaultRegistry()
	//  modelInfoCoderRegistry.GetModelSettings()
	modelCoder, err := modelinfocoder.NewModel(model, nil, nil, "")
	if err != nil {
		logger.Error("Error creating model coder", "error", err)
		os.Exit(1)
	}

	fm := filemanager.NewLocalFileManager()
	// Setup chat parameters
	_ = chat.NewChatParams()
	rootPath, err := os.Getwd()
	if err != nil {
		logger.Error("Error getting current working directory", "error", err)
		os.Exit(1)
	}
	logger.Info("rootPath", "rootPath", rootPath)

	llmClient, err := llmclient.New(modelInfo.Provider)
	if err != nil {
		logger.Error("Error creating llm client", "error", err)
		os.Exit(1)
	}

	gitRepo, err := gitrepo.NewGitRepo(logger, nil, rootPath)
	if err != nil {
		logger.Error("Error creating git repo", "error", err)
	}

	rm := repomanager.NewRepoManager(rootPath, logger, fm, gitRepo)

	promptName := "EditBlock"
	pts := prompts.New(promptName)
	if pts == nil {
		logger.Error("Error creating prompts", "prompt_name", promptName)
		os.Exit(1)
	}

	h := highlighter.NewHighlighter(os.Stdout)
	coderSettings := settings.NewCoderSettings(modelCoder)

	coderOpts := coder.CoderOptions{
		Logger:       logger,
		LlmClient:    llmClient,
		RepoManager:  rm,
		Prompts:      pts,
		StreamWriter: h,
		Settings:     coderSettings,
	}

	coder := coder.NewBaseCoder(coderOpts)

	console := consolecoder.New(coder, h)
	console.Run()
}
