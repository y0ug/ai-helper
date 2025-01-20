package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/reeflective/console"
)

func main() {
}

func StartConsole() {
	app := console.New("example")

	store := make(map[string]string)

	app.NewlineBefore = true
	app.NewlineAfter = true
	menu := app.ActiveMenu()
	menu.AddInterrupt(io.EOF, exitCtrlD)
	menu.SetCommands(mainMenuCommands(app, store))
	app.Start()
}

func exitCtrlD(c *console.Console) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Confirm exit (Y/y): ")
	text, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(text)

	if (answer == "Y") || (answer == "y") {
		os.Exit(0)
	}
}
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/llmagent"
	"github.com/y0ug/ai-helper/internal/io"
	"github.com/y0ug/ai-helper/pkg/highlighter"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
	"github.com/y0ug/ai-helper/pkg/llmclient/modelinfo"
)

func generateSessionID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

func main() {
	configFile := flag.String("config", "", "Config file path")
	verbose := flag.Bool("v", false, "Show verbose output")
	attachFiles := flag.String("files", "", "Comma-separated list of files to attach")
	flag.Parse()

	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}

	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      level,
		TimeFormat: time.Kitchen,
	}))

	// Load configuration
	cfgPath := *configFile
	if cfgPath == "" {
		var err error
		cfgPath, err = io.FindConfigFile(".")
		if err != nil {
			logger.Error("Error finding config file", "error", err)
			os.Exit(1)
		}
	}

	loader := config.NewLoader()
	cfg, err := loader.Load(cfgPath)
	if err != nil {
		logger.Error("Error loading config", "error", err)
		os.Exit(1)
	}

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

	// Get command and args
	args := flag.Args()
	if len(args) < 1 {
		logger.Error("Command required")
		os.Exit(1)
	}

	command := args[0]
	cmd, ok := cfg.Commands[command]
	if !ok {
		logger.Error("Unknown command", "command", command)
		os.Exit(1)
	}

	// Setup chat parameters
	chatParams := chat.NewChatParams()
	if model := os.Getenv("AI_MODEL"); model != "" {
		chatParams.Model = model
	}

	// Create template agent
	agent, err := llmagent.NewTemplateAgent(
		generateSessionID(),
		logger,
		&cmd,
		chatParams,
		infoProviders,
		&cfg.MCPServers,
	)
	if err != nil {
		logger.Error("Error creating agent", "error", err)
		os.Exit(1)
	}

	// Load input arguments
	inputArgs := args[1:]
	argsMap := make(map[string]string)
	argsMap["Input"] = strings.Join(inputArgs, " ")
	if err := agent.LoadArgs(argsMap); err != nil {
		logger.Error("Error loading args", "error", err)
		os.Exit(1)
	}

	// Load additional files if specified
	if *attachFiles != "" {
		files := strings.Split(*attachFiles, ",")
		for _, file := range files {
			file = strings.TrimSpace(file)
			if err := agent.LoadFiles(file); err != nil {
				logger.Error("Error loading file", "file", file, "error", err)
				os.Exit(1)
			}
		}
	}

	// Execute the command
	h := highlighter.NewHighlighter(os.Stdout)
	_, cost, err := agent.Execute(context.Background(), h)
	if err != nil {
		logger.Error("Error executing command", "error", err)
		os.Exit(1)
	}

	if *verbose {
		logger.Debug("Command completed", "cost", cost)
	}
}
