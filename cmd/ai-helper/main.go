package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/y0ug/ai-helper/internal/ai"
	"github.com/y0ug/ai-helper/internal/chat"
	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/io"
	"github.com/y0ug/ai-helper/internal/stats"
	"github.com/y0ug/ai-helper/internal/version"
)

func main() {
	// Parse command line flags
	outputFile := flag.String("output", "", "Output file path")
	configFile := flag.String("config", "", "Config file path")
	showStats := flag.Bool("stats", false, "Show usage statistics")
	showList := flag.Bool("list", false, "List available commands")
	verbose := flag.Bool("v", false, "Show verbose cost information")
	genCompletion := flag.String("completion", "", "Generate shell completion script (zsh|bash)")
	showPrompt := flag.Bool("show-prompt", false, "Show only the generated prompt")
	attachFiles := flag.String("files", "", "Comma-separated list of files to attach")
	showVersion := flag.Bool("version", false, "Show version information")
	interactiveMode := flag.Bool("i", false, "Interactive chat mode")
	flag.Parse()

	// Create AI client early as it's needed for multiple features
	client, err := ai.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating AI client: %v\n", err)
		os.Exit(1)
	}

	// Handle version display
	if *showVersion {
		fmt.Printf("ai-helper version %s\n", version.Version)
		fmt.Printf("Commit: %s\n", version.CommitHash)
		fmt.Printf("Built: %s\n", version.BuildDate)
		os.Exit(0)
	}

	// Handle interactive mode
	if *interactiveMode {
		command := ""
		if len(flag.Args()) > 0 {
			command = flag.Args()[0]
		}

		var initialPrompt, systemPrompt string
		if command != "" {
			// Get command configuration
			cmd, ok := cfg.Commands[command]
			if !ok {
				fmt.Fprintf(os.Stderr, "Error: Unknown command '%s'\n", command)
				os.Exit(1)
			}

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
			}
		}

		chatSession, err := chat.NewChat(client, "gpt-4")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing chat: %v\n", err)
			os.Exit(1)
		}

		if initialPrompt != "" {
			chatSession.SetInitialPrompt(initialPrompt)
		}
		if systemPrompt != "" {
			chatSession.SetSystemPrompt(systemPrompt)
		}

		if err := chatSession.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error in chat mode: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Handle completion script generation
	if *genCompletion != "" {
		switch *genCompletion {
		case "zsh":
			fmt.Println(generateZshCompletion())
			os.Exit(0)
		case "bash":
			fmt.Println(generateBashCompletion())
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

	loader := config.NewLoader()
	cfg, err := loader.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

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
		tracker, err := stats.NewTracker()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating stats tracker: %v\n", err)
			os.Exit(1)
		}

		stats := tracker.GetStats()
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

	// Get the command and remaining args
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: Command required")
		os.Exit(1)
	}
	command := args[0]
	inputArgs := args[1:]

	// Create config loader and load config
	if err := cfg.ValidateConfig(); err != nil {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
	}

	// Get command configuration
	cmd, ok := cfg.Commands[command]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: Unknown command '%s'\n", command)
		os.Exit(1)
	}

	// Prepare input configuration
	var inputTypes []string
	var fallbackCmd string

	// Look for input variable configuration
	for _, v := range cmd.Variables {
		if v.Name == "Input" && v.Type != "" {
			// Split type string in case it contains multiple types (e.g. "stdin|arg")
			inputTypes = strings.Split(v.Type, "|")
			fallbackCmd = v.Exec
			break
		}
	}

	// Read input only if command requires it
	var input string
	if cmd.Input {
		var err error
		input, err = io.ReadInput(inputArgs, inputTypes, fallbackCmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}
	}


	// Load prompt and system prompt content and variables
	promptContent, systemContent, vars, err := config.LoadPromptContent(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading prompt: %v\n", err)
		os.Exit(1)
	}

	// Prepare template data with input, variables and environment
	templateData := map[string]interface{}{
		"Input": input,
		"env":   make(map[string]string),
		"Files": make(map[string]string),
	}
	// Add environment variables
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			templateData["env"].(map[string]string)[pair[0]] = pair[1]
		}
	}
	// Load file contents from both config and command line
	filesToLoad := make(map[string]bool)

	// Add files from config
	for _, filepath := range cmd.Files {
		filesToLoad[filepath] = true
	}

	// Add files from command line flag
	if *attachFiles != "" {
		for _, filepath := range strings.Split(*attachFiles, ",") {
			filesToLoad[strings.TrimSpace(filepath)] = true
		}
	}

	// Add any variables from command config
	for k, v := range vars {
		templateData[k] = v
	}

	// Create template functions
	funcMap := template.FuncMap{
		"fileContent": func(path string) string {
			content, ok := templateData["Files"].(map[string]string)[path]
			if !ok {
				return fmt.Sprintf("Error: file %s not found", path)
			}
			return content
		},
		"fileExt": filepath.Ext,
		"fileName": func(path string) string {
			return filepath.Base(path)
		},
		"formatFile": func(path string) string {
			content, ok := templateData["Files"].(map[string]string)[path]
			if !ok {
				return fmt.Sprintf("Error: file %s not found", path)
			}
			ext := filepath.Ext(path)
			return fmt.Sprintf("```%s\n%s\n```", ext[1:], content)
		},
	}

	// Load all unique files
	for filepath := range filesToLoad {
		content, err := os.ReadFile(filepath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", filepath, err)
			os.Exit(1)
		}
		templateData["Files"].(map[string]string)[filepath] = string(content)

	}

	// Parse and execute the prompt and system templates
	tmpl, err := template.New("prompt").Funcs(funcMap).Parse(promptContent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing prompt template: %v\n", err)
		os.Exit(1)
	}

	var promptBuf bytes.Buffer
	if err := tmpl.Execute(&promptBuf, templateData); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing prompt template: %v\n", err)
		os.Exit(1)
	}

	var systemBuf bytes.Buffer
	if systemContent != "" {
		systemTmpl, err := template.New("system").Funcs(funcMap).Parse(systemContent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing system template: %v\n", err)
			os.Exit(1)
		}

		if err := systemTmpl.Execute(&systemBuf, templateData); err != nil {
			fmt.Fprintf(os.Stderr, "Error executing system template: %v\n", err)
			os.Exit(1)
		}
	}

	// If show-prompt flag is set, print the prompt and exit
	if *showPrompt {
		fmt.Println(promptBuf.String())
		os.Exit(0)
	}

	resp, err := client.Generate(promptBuf.String(), systemBuf.String(), command)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating response: %v\n", err)
		os.Exit(1)
	}

	// Print token usage and cost to stderr
	if *verbose {
		fmt.Fprintf(
			os.Stderr,
			"Tokens - Input: %d, Output: %d\n",
			resp.InputTokens,
			resp.OutputTokens,
		)
		fmt.Fprintf(os.Stderr, "Estimated cost: $%.4f\n", resp.Cost)
	}

	// Ensure output directory exists if writing to file
	if *outputFile != "" {
		if err := io.EnsureDirectory(*outputFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
			os.Exit(1)
		}
	}

	// Write output
	if err := io.WriteOutput(resp.Content, *outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}
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
