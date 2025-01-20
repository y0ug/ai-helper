package console

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/y0ug/ai-helper/internal/llmagent"
	"github.com/y0ug/ai-helper/pkg/highlighter"
)

type Console struct {
	agent         *llmagent.TemplateAgent
	h             *highlighter.Highlighter
	commands      map[string]Command
	pt            *prompt.Prompt
	attachedFiles []string
}

// Command represents a chat command
type Command struct {
	name        string
	description string
	handler     func(args []string)
}

func New(agent *llmagent.TemplateAgent) *Console {
	c := &Console{
		agent: agent,
		h:     highlighter.NewHighlighter(os.Stdout),
	}
	c.commands = map[string]Command{
		"/help": {
			name:        "help",
			description: "Show available commands",
			handler:     c.handleHelp,
		},
		"/quit": {
			name:        "quit",
			description: "Exit the chat",
			handler:     c.handleQuit,
		},
		"/add": {
			name:        "add",
			description: "Add file(s) to the conversation",
			handler:     c.handleAddFile,
		},
		"/remove": {
			name:        "remove",
			description: "Remove file(s) from the conversation",
			handler:     c.handleRemoveFile,
		},
		"/files": {
			name:        "files",
			description: "List currently attached files",
			handler:     c.handleListFiles,
		},
	}

	c.pt = prompt.New(
		c.executor,
		c.completer,
		prompt.OptionLivePrefix(c.UpdatePrompt),
		prompt.OptionTitle("Chat"),
		prompt.OptionPrefix(fmt.Sprintf("%s ➜ ", agent.ModelInfo.Name)),
		prompt.OptionInputTextColor(prompt.Yellow),
		prompt.OptionPrefixTextColor(prompt.Blue),
		prompt.OptionMaxSuggestion(5),
		prompt.OptionHistory([]string{}),
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlC,
			Fn:  func(*prompt.Buffer) { c.handleQuit(nil) },
		}),
	)

	return c
}

func (c *Console) UpdatePrompt() (string, bool) {
	return fmt.Sprintf(
		"%s (%s,%s) ➜ ",
		c.agent.ModelInfo.Name,
		c.agent.Command.Name,
		c.agent.ConversationManager.CurrentState,
	), true
}

func (c *Console) Run() {
	if !c.agent.ConversationManager.IsInputNeeded() {
		c.agent.Execute(context.Background(), c.h)
	}
	c.pt.Run()
}

func (c *Console) handleHelp(args []string) {
	fmt.Println("Available commands:")
	for _, cmd := range c.commands {
		fmt.Printf("/%s - %s\n", cmd.name, cmd.description)
	}
}

func (c *Console) handleQuit(args []string) {
	fmt.Println("Goodbye!")
	// You might want to cleanup here
	// Example: close connections, save state, etc.
	panic("quit") // Quick way to exit, you might want to handle this more gracefully
}

func (c *Console) completer(d prompt.Document) []prompt.Suggest {
	var suggestions []prompt.Suggest

	word := d.GetWordBeforeCursor()

	// If the word starts with /, suggest commands
	if strings.HasPrefix(word, "/") {
		for cmdName, cmd := range c.commands {
			suggestions = append(suggestions, prompt.Suggest{
				Text:        cmdName,
				Description: cmd.description,
			})
		}
	}

	return prompt.FilterHasPrefix(suggestions, word, true)
}

func (c *Console) handleAddFile(args []string) {
	if len(args) == 0 {
		fmt.Println("Please specify file(s) to add")
		return
	}

	for _, file := range args {
		if err := c.agent.ConversationManager.LoadFiles(file); err != nil {
			fmt.Printf("Error loading file %s: %v\n", file, err)
			continue
		}
		fmt.Printf("Added file: %s\n", file)
	}
}

func (c *Console) handleRemoveFile(args []string) {
	if len(args) == 0 {
		fmt.Println("Please specify file(s) to remove")
		return
	}

	loadedFiles := c.agent.ConversationManager.GetLoadedFiles()
	for _, file := range args {
		found := false
		for _, loaded := range loadedFiles {
			if loaded == file {
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("File not found: %s\n", file)
		} else {
			c.agent.ConversationManager.RemoveFiles(file)
			fmt.Printf("Removed file: %s\n", file)
		}
	}
}

func (c *Console) handleListFiles(args []string) {
	files := c.agent.ConversationManager.GetLoadedFiles()
	if len(files) == 0 {
		fmt.Println("No files currently attached")
		return
	}
	fmt.Println("Currently attached files:")
	for _, file := range files {
		fmt.Printf("- %s\n", file)
	}
}

func (c *Console) executor(input string) {
	input = strings.TrimSpace(input)

	if input == "" {
		return
	}

	// Handle commands
	if strings.HasPrefix(input, "/") {
		parts := strings.Fields(input)
		cmd, exists := c.commands[parts[0]]
		if exists {
			cmd.handler(parts[1:])
			return
		}
		fmt.Printf("Unknown command: %s\n", parts[0])
		return
	}

	// Add the command to the agent's message queue
	c.agent.SetInput(input)

	// // Load any attached files before sending
	// for _, file := range c.attachedFiles {
	// 	if err := c.agent.LoadFiles(file); err != nil {
	// 		fmt.Printf("Error loading file %s: %v\n", file, err)
	// 		return
	// 	}
	// }

	// Send the request to the AI agent
	responses, cost, err := c.agent.Execute(context.Background(), c.h)
	// responses, cost, err := c.agent.ConversationManager.Execute(context.Background(), c.h)
	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") {
			fmt.Println("❌ Request timed out. The model took too long to respond.")
		} else if strings.Contains(err.Error(), "rate limit") {
			fmt.Println("❌ Rate limit exceeded. Please wait a moment before trying again.")
		} else {
			fmt.Printf("❌ Error: %v\n", err)
		}
		return
	}

	if len(responses) == 0 {
		fmt.Println("⚠️ Warning: No response received from the model")
		return
	}

	if cost > 0 {
		fmt.Printf("💰 Cost: $%.4f\n", cost)
	}
}
