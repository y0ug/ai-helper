package console

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/y0ug/ai-helper/internal/llmagent"
	"github.com/y0ug/ai-helper/pkg/highlighter"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
)

type Console struct {
	agent    *llmagent.TemplateAgent
	h        *highlighter.Highlighter
	commands map[string]Command
	pt       *prompt.Prompt
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
	}

	c.pt = prompt.New(
		c.executor,
		c.completer,
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

func (c *Console) Run() {
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
	c.agent.AddMessage(chat.NewUserMessage(input))

	// Send the request to the AI agent
	_, _, err := c.agent.Do(context.Background(), c.h)
	if err != nil {
		fmt.Printf("Error generating response: %v\n", err)
		return
	}
}
