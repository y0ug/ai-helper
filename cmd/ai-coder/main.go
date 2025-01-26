package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/lmittmann/tint"
	"github.com/y0ug/ai-helper/internal/assistant"
	modelinfocoder "github.com/y0ug/ai-helper/internal/assistant/llm/models"
	"github.com/y0ug/ai-helper/internal/assistant/prompt/prompts"
	"github.com/y0ug/ai-helper/internal/assistant/repomanager"
	"github.com/y0ug/ai-helper/internal/assistant/settings"
	"github.com/y0ug/ai-helper/internal/assistant/ui"
	"github.com/y0ug/ai-helper/internal/consolecoder"
	"github.com/y0ug/ai-helper/internal/filemanager"
	"github.com/y0ug/ai-helper/pkg/gitrepo"
	"github.com/y0ug/ai-helper/pkg/highlighter"
	"github.com/y0ug/ai-helper/pkg/llmhaven"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
	"github.com/y0ug/ai-helper/pkg/llmhaven/http/options"
	"github.com/y0ug/ai-helper/pkg/llmhaven/modelinfo"
)

func main() {
	ctx := context.Background()
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

	// Setup the model info provider with a cache json file
	infoProviderCacheFile := filepath.Join(configDir, "provider_cache.json")
	modelInfoProvider, err := modelinfo.New(ctx, infoProviderCacheFile)
	if err != nil {
		logger.Error("Error creating model info providers", "error", err)
		os.Exit(1)
	}

	// Get settings from environment variables
	model := os.Getenv("AI_MODEL")
	if model == "" {
		logger.Error("AI_MODEL environment variable not set")
		os.Exit(1)
	}

	promptName := os.Getenv("AI_PROMPT")
	if promptName == "" {
		promptName = "EditBlock"
	}

	// Find model info and provider from the model string
	modelInfo, err := modelinfo.Get(model, modelInfoProvider)
	if err != nil {
		logger.Error("Error parsing model info", "error", err)
		os.Exit(1)
	}

	// Define the model with the settings need for the assistant
	modelCoder, err := modelinfocoder.NewModel(*modelInfo, nil, nil, "")
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
	requestOpts := []options.RequestOption{
		// options.WithMiddleware(middleware.LoggingMiddleware()),
		// options.WithMiddleware(middleware.TimeitMiddleware(logger)),
	}
	llmClient, err := llmhaven.New(modelInfo.Provider, requestOpts...)
	if err != nil {
		logger.Error("Error creating llm client", "error", err)
		os.Exit(1)
	}

	gitRepo, err := gitrepo.NewGitRepo(logger, nil, rootPath)
	if err != nil {
		logger.Error("Error creating git repo", "error", err)
	}

	rm := repomanager.NewRepoManager(rootPath, logger, fm, gitRepo)

	pts := prompts.New(promptName)
	if pts == nil {
		logger.Error("Error creating prompts", "prompt_name", promptName)
		os.Exit(1)
	}

	h := highlighter.NewHighlighter(os.Stdout)
	coderSettings := settings.NewCoderSettings(modelCoder)

	uim := ui.NewUIInteractionManager(h)

	// Capture Ctrl-C (SIGINT)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		uim.Shutdown()
	}()

	coderOpts := assistant.AssistantOptions{
		Logger:      logger,
		LlmClient:   llmClient,
		RepoManager: rm,
		Prompts:     pts,
		// StreamWriter: h,
		Settings: coderSettings,
		Stream:   true,
		// Uim:          uim,
	}
	coder := assistant.NewAssistantOrchestrator(coderOpts)

	coder.Start(ctx)
	defer coder.Stop()

	console := consolecoder.New(coder, h, uim, nil)
	console.Run()
}
