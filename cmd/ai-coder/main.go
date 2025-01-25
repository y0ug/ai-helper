package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/y0ug/ai-helper/internal/assistant"
	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/actions/executors"
	modelinfocoder "github.com/y0ug/ai-helper/internal/assistant/llm/models"
	"github.com/y0ug/ai-helper/internal/assistant/prompts"
	"github.com/y0ug/ai-helper/internal/assistant/repomanager"
	"github.com/y0ug/ai-helper/internal/assistant/settings"
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

	promptName := os.Getenv("AI_PROMPT")
	if promptName == "" {
		promptName = "EditBlock"
	}

	modelInfo, err := modelinfo.Parse(model, infoProviders)
	if err != nil {
		logger.Error("Error parsing model info", "error", err)
		os.Exit(1)
	}
	// modelInfoCoderRegistry := modelinfocoder.InitializeDefaultRegistry()
	//  modelInfoCoderRegistry.GetModelSettings()
	modelCoder, err := modelinfocoder.NewModel(modelInfo.Name, nil, nil, "")
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

	// Create channels
	responseChan := make(chan executors.UserResponse, 10)
	confirmChan := make(chan actions.Action, 10)

	// Start user input handler
	go func() {
		for confirmAction := range confirmChan {
			// Present confirmation to user
			fmt.Printf(
				"Allow command: %s? (y/n): ",
				confirmAction.Payload.(actions.UserConfirmAction).Question,
			)

			// Get user response
			var response string
			fmt.Scanln(&response)

			// Send response
			responseChan <- executors.UserResponse{
				Allowed:    strings.ToLower(response) == "y",
				ParentID:   confirmAction.Context.ParentID.String(),
				ChainID:    confirmAction.Context.ChainID.String(),
				ToolCallID: confirmAction.Context.ToolCallID,
			}
		}
	}()

	coderOpts := assistant.AssistantOptions{
		Logger:       logger,
		LlmClient:    llmClient,
		RepoManager:  rm,
		Prompts:      pts,
		StreamWriter: h,
		Settings:     coderSettings,
		Stream:       true,
		ResponseChan: responseChan,
		ConfirmChan:  confirmChan,
	}
	coder := assistant.NewAssistantOrchestrator(coderOpts)

	console := consolecoder.New(coder, h)
	console.Run()
}
