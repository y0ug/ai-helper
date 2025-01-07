package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/y0ug/ai-helper/internal/ai"
	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/io"
	"github.com/y0ug/ai-helper/internal/stats"
)

func main() {
	// Parse command line flags
	outputFile := flag.String("output", "", "Output file path")
	configFile := flag.String("config", "", "Config file path")
	showStats := flag.Bool("stats", false, "Show usage statistics")
	showList := flag.Bool("list", false, "List available commands")
	verbose := flag.Bool("v", false, "Show verbose cost information")
	genCompletion := flag.String("completion", "", "Generate shell completion script (zsh)")
	flag.Parse()

	// Handle completion script generation
	if *genCompletion != "" {
		switch *genCompletion {
		case "zsh":
			fmt.Println(generateZshCompletion())
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

	// Read input using configured methods
	input, err := io.ReadInput(inputArgs, inputTypes, fallbackCmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	// Create AI client
	client, err := ai.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating AI client: %v\n", err)
		os.Exit(1)
	}

	// Load prompt content and variables
	promptContent, vars, err := config.LoadPromptContent(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading prompt: %v\n", err)
		os.Exit(1)
	}

	// Prepare template data with input, variables and environment
	templateData := map[string]interface{}{
		"Input": input,
		"env":   make(map[string]string),
	}
	// Add environment variables
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			templateData["env"].(map[string]string)[pair[0]] = pair[1]
		}
	}
	// Add any variables from command config
	for k, v := range vars {
		templateData[k] = v
	}

	// Parse and execute the prompt template
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

	// fmt.Println("promptBuf: ", promptBuf.String())
	resp, err := client.Generate(promptBuf.String(), command)
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
