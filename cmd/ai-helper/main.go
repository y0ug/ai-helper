package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/console"
	"github.com/y0ug/ai-helper/internal/io"
	"github.com/y0ug/ai-helper/internal/llmagent"
	"github.com/y0ug/ai-helper/internal/stats"
	"github.com/y0ug/ai-helper/internal/version"
	"github.com/y0ug/ai-helper/pkg/highlighter"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
	"github.com/y0ug/ai-helper/pkg/llmclient/modelinfo"
)

const (
	EnvAIModel = "AI_MODEL"
)

func generateSessionID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

func GetEnvAIModel(infoProviders modelinfo.Provider) (*modelinfo.Model, error) {
	modelStr := os.Getenv(EnvAIModel)
	if modelStr == "" {
		return nil, fmt.Errorf("AI_MODEL environment variable not set")
	}

	model, err := modelinfo.Parse(modelStr, infoProviders)
	if err != nil {
		return nil, fmt.Errorf("failed to parse model: %w", err)
	}

	return model, nil
}

func main() {
	// Parse command line flags
	// outputFile := flag.String("output", "", "Output file path")
	configFile := flag.String("config", "", "Config file path")
	showStats := flag.Bool("stats", false, "Show usage statistics")
	showList := flag.Bool("list", false, "List available commands")
	verbose := flag.Bool("v", false, "Show verbose cost information")
	genCompletion := flag.String("completion", "", "Generate shell completion script (zsh|bash)")
	// showPrompt := flag.Bool("show-prompt", false, "Show only the generated prompt")
	attachFiles := flag.String("files", "", "Comma-separated list of files to attach")
	showVersion := flag.Bool("version", false, "Show version information")
	interactiveMode := flag.Bool("i", false, "Interactive chat mode")
	flag.Parse()
	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      level,
		TimeFormat: time.Kitchen,
	}))

	//    &slog.HandlerOptions{
	// 	Level: level,
	// }))

	// Create AI client early as it's needed for multiple features
	configDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get home directory: %s", err)
		os.Exit(1)
	}

	configDir = filepath.Join(configDir, ".config", "ai-helper")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create config directory: %s", err)
		os.Exit(1)
	}

	cacheDir := io.GetCacheDir()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create cache directory: %w", err)
		os.Exit(1)
	}
	statsTracker, err := stats.NewTracker(cacheDir)

	infoProviderCacheFile := filepath.Join(configDir, "provider_cache.json")
	infoProviders, err := modelinfo.New(infoProviderCacheFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating info providers: %v\n", err)
		os.Exit(1)
	}
	model, err := GetEnvAIModel(infoProviders)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting model: %v\n", err)
		os.Exit(1)
	}

	// Handle version display
	if *showVersion {
		fmt.Printf("ai-helper version %s\n", version.Version)
		fmt.Printf("Commit: %s\n", version.CommitHash)
		fmt.Printf("Built: %s\n", version.BuildDate)
		os.Exit(0)
	}

	// Handle completion script generation
	if *genCompletion != "" {
		switch *genCompletion {
		case "zsh":
			// fmt.Println(generateZshCompletion())
			os.Exit(0)
		case "bash":
			// fmt.Println(generateBashCompletion())
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "Unsupported shell for completion: %s\n", *genCompletion)
			os.Exit(1)
		}
	}

	// Load configuration early for list command
	cfgPath := *configFile
	if cfgPath == "" {
		var err error
		cfgPath, err = io.FindConfigFile(".")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
	ctx := context.Background()

	loader := config.NewLoader()
	cfg, err := loader.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	toolProcessor := llmagent.NewToolProcessor(logger)
	toolProcessor.Start(ctx, &cfg.MCPServers)

	// Handle list command
	if *showList {
		fmt.Println("Available commands:")
		for name, cmd := range cfg.Commands {
			if cmd.Description != "" {
				fmt.Printf("  %-15s %s\n", name, cmd.Description)
			} else {
				fmt.Printf("  %s\n", name)
			}
		}
		os.Exit(0)
	}

	// Handle stats display
	if *showStats {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating stats tracker: %v\n", err)
			os.Exit(1)
		}

		stats := statsTracker.GetStats()
		fmt.Println("AI Provider Usage Statistics:")
		fmt.Println("============================")

		if len(stats) == 0 {
			fmt.Println("No statistics available yet.")
			os.Exit(0)
		}

		for provider, pStats := range stats {
			fmt.Printf("\nProvider: %s\n", provider)
			fmt.Printf("  Queries:        %d\n", pStats.Queries)
			fmt.Printf("  Input Tokens:   %d\n", pStats.InputTokens)
			fmt.Printf("  Output Tokens:  %d\n", pStats.OutputTokens)
			fmt.Printf("  Total Cost:     $%.4f\n", pStats.Cost)
			fmt.Printf("  Last Used:      %s\n", pStats.LastUsed.Format("2006-01-02 15:04:05"))

			if len(pStats.Commands) > 0 {
				fmt.Printf("\n  Commands:\n")
				for cmd, cmdStats := range pStats.Commands {
					fmt.Printf("    %s:\n", cmd)
					fmt.Printf("      Count:         %d\n", cmdStats.Count)
					fmt.Printf("      Input Tokens:  %d\n", cmdStats.InputTokens)
					fmt.Printf("      Output Tokens: %d\n", cmdStats.OutputTokens)
					fmt.Printf("      Total Cost:    $%.4f\n", cmdStats.Cost)
					fmt.Printf(
						"      Last Used:     %s\n",
						cmdStats.LastUsed.Format("2006-01-02 15:04:05"),
					)
				}
			}
		}
		os.Exit(0)
	}

	chatParams := chat.NewChatParams(
		chat.WithModel(model.Name),
		chat.WithMaxTokens(model.Metadata.MaxTokens),
	)

	command := ""
	var args []string
	if len(flag.Args()) > 0 {
		args = flag.Args()
		command = args[0]
	}

	command = strings.TrimSpace(command)
	var initialPrompt, systemPrompt string
	if command == "" {
		logger.Error("Error: Command required")
		os.Exit(0)
	}
	// Get command configuration
	cmd, ok := cfg.Commands[command]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: Unknown command '%s'\n", command)
		os.Exit(1)
	}
	agent, err := llmagent.NewTemplateAgent(
		generateSessionID(),
		logger,
		&cmd,
		chatParams,
		infoProviders,
		toolProcessor,
	)
	if err != nil {
		logger.Error("failed to create agent", "error", err)
		return
	}
	// Handle interactive mode
	if *interactiveMode {
		// Create an agent for this command

		// Load prompt and system prompt content
		promptContent, systemContent, vars, err := config.LoadPromptContent(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading prompt: %v\n", err)
			os.Exit(1)
		}

		// Process the prompts with template data
		templateData := map[string]interface{}{
			"env":   make(map[string]string),
			"Files": make(map[string]string),
		}
		// Add variables from command config
		for k, v := range vars {
			templateData[k] = v
		}

		// Parse and execute the prompts
		tmpl, err := template.New("prompt").Parse(promptContent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing prompt template: %v\n", err)
			os.Exit(1)
		}

		var promptBuf bytes.Buffer
		if err := tmpl.Execute(&promptBuf, templateData); err != nil {
			fmt.Fprintf(os.Stderr, "Error executing prompt template: %v\n", err)
			os.Exit(1)
		}
		initialPrompt = promptBuf.String()

		if systemContent != "" {
			systemTmpl, err := template.New("system").Parse(systemContent)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing system template: %v\n", err)
				os.Exit(1)

			}
			var systemBuf bytes.Buffer
			if err := systemTmpl.Execute(&systemBuf, templateData); err != nil {
				fmt.Fprintf(os.Stderr, "Error executing system template: %v\n", err)
				os.Exit(1)
			}
			systemPrompt = systemBuf.String()
			if systemPrompt != "" {
				agent.AddMessage(chat.NewSystemMessage(systemPrompt))
			}
		}
		if initialPrompt != "" {
			agent.AddMessage(chat.NewUserMessage(initialPrompt))
		}

		console := console.New(agent)
		console.Run()
		return
	}

	inputArgs := args[1:]
	h := highlighter.NewHighlighter(os.Stdout)
	agent.GetConversation().GetCtx().SetVariable("Input", strings.Join(inputArgs, " "))

	if *attachFiles != "" {
		additionalFiles := strings.Split(*attachFiles, ",")
		for _, filepath := range additionalFiles {
			filepath = strings.TrimSpace(filepath)
			logger.Debug("Loading file", "path", filepath)
			if err := agent.GetConversation().GetCtx().GetFM().AddFile(filepath, false); err != nil {
				logger.Error("failed to load files", "Error", err)
				return
			}
		}
	}

	_, cost, err := agent.Execute(context.Background(), h)
	if err != nil {
		logger.Error("failed to execute agent", "Error", err)
		return
	}

	// logger.Debug("Message", "msg", msg)
	logger.Debug("Cost", "cost", cost)

	// // Get the command and remaining args
	// args := flag.Args()
	// if len(args) < 1 {
	// 	fmt.Fprintln(os.Stderr, "Error: Command required")
	// 	os.Exit(1)
	// }
	// command := args[0]
	// inputArgs := args[1:]
	//
	// // Create config loader and load config
	// if err := cfg.ValidateConfig(); err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
	// 	os.Exit(1)
	// }
	//
	// // Get command configuration
	// cmd, ok := cfg.Commands[command]
	// if !ok {
	// 	fmt.Fprintf(os.Stderr, "Error: Unknown command '%s'\n", command)
	// 	os.Exit(1)
	// }
	//
	// // Prepare input configuration
	// var inputTypes []string
	// var fallbackCmd string
	//
	// // Look for input variable configuration
	// for _, v := range cmd.Variables {
	// 	if v.Name == "Input" && v.Type != "" {
	// 		// Split type string in case it contains multiple types (e.g. "stdin|arg")
	// 		inputTypes = strings.Split(v.Type, "|")
	// 		fallbackCmd = v.Exec
	// 		break
	// 	}
	// }
	//
	// // Read input only if command requires it
	// var input string
	// if cmd.Input {
	// 	var err error
	// 	input, err = io.ReadInput(inputArgs, inputTypes, fallbackCmd)
	// 	if err != nil {
	// 		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
	// 		os.Exit(1)
	// 	}
	// 	agent.TemplateData.Input = input
	// }
	//
	// // Load command configuration into agent
	// if err := agent.LoadCommand(&cmd); err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error loading command: %v\n", err)
	// 	os.Exit(1)
	// }
	//
	// Add files from command line flag

	//
	// // Apply the command with input
	// if err := agent.ApplyCommand(input); err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error applying command: %v\n", err)
	// 	os.Exit(1)
	// }
	//
	// // If show-prompt flag is set, print the last user message and exit
	// if *showPrompt {
	// 	msgs := agent.GetMessages()
	// 	for _, v := range msgs {
	// 		fmt.Printf("%s: %s\n", v.GetRole(), v.GetContent())
	// 	}
	// 	os.Exit(1)
	// }
	//
	// // Generate response using the agent
	// _, responses, err := agent.SendRequest()
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error generating response: %v\n", err)
	// 	os.Exit(1)
	// }
	//
	// for _, resp := range responses {
	// 	// Print token usage and cost to stderr
	// 	if *verbose {
	// 		fmt.Fprintf(
	// 			os.Stderr,
	// 			"Tokens - Input: %d, Output: %d\n",
	// 			resp.GetUsage().GetInputTokens(),
	// 			resp.GetUsage().GetOutputTokens(),
	// 		)
	// 	}
	// 	cost := fmt.Sprintf("$%.4f", resp.GetUsage().GetCost())
	// 	fmt.Fprintf(
	// 		os.Stderr,
	// 		"Session: %s | Model: %s | Estimated cost: %s\n",
	// 		agent.ID,
	// 		agent.Model.Name,
	// 		cost,
	// 	)
	//
	// 	// Ensure output directory exists if writing to file
	// 	if *outputFile != "" {
	// 		if err := io.EnsureDirectory(*outputFile); err != nil {
	// 			fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
	// 			os.Exit(1)
	// 		}
	// 	}
	//
	// 	// Write output
	// 	if err := io.WriteOutput(resp.GetChoice().GetMessage().GetContent().String(), *outputFile); err != nil {
	// 		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
	// 		os.Exit(1)
	// 	}
	// }
	//
	// agent.Save()
}

func generateBashCompletion() string {
	return `_ai_helper() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="-output -config -stats -list -v -completion -show-prompt -files -version -i"

    if [[ ${cur} == -* ]] ; then
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi
}
complete -F _ai_helper ai-helper`
}
