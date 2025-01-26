package console

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/actions/executors"
	"github.com/y0ug/ai-helper/internal/llmagent"
	"github.com/y0ug/ai-helper/pkg/highlighter"
)

type Console struct {
	agent       *llmagent.TemplateAgent
	h           *highlighter.Highlighter
	commands    map[string]Command
	pt          *prompt.Prompt
	historyFile string
}

func getHistoryFilePath() string {
	usr, err := user.Current()
	if err != nil {
		return ".ai-helper-history"
	}
	return filepath.Join(usr.HomeDir, ".ai-helper-history")
}

// Command represents a chat command
type Command struct {
	name        string
	description string
	handler     func(args []string)
}

func New(agent *llmagent.TemplateAgent) *Console {
	c := &Console{
		coder:        coder,
		h:            h,
		historyFile:  getHistoryFilePath(),
		statusChan:   make(chan string),
		inputChan:    make(chan string),
		confirmChan:  make(chan actions.Action),
		responseChan: make(chan executors.UserResponse),
	}

	c.setCommands()

	c.pt = prompt.New(
		c.executor,
		c.completer,
		prompt.OptionLivePrefix(c.UpdatePrompt),
		prompt.OptionTitle("Chat"),
		prompt.OptionPrefix(fmt.Sprintf(" Γ₧£ ")),
		prompt.OptionInputTextColor(prompt.Yellow),
		prompt.OptionPrefixTextColor(prompt.Blue),
		prompt.OptionMaxSuggestion(5),
		prompt.OptionHistory(c.loadHistory()),
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
		c.agent.GetModelName(),
		c.agent.GetConversation().GetCtx().GetCommand().Name,
		c.agent.GetConversation().GetCurrentState(),
	), true
}

func (c *Console) Run() {
	if !c.agent.GetConversation().IsInputNeeded() {
		// c.agent.Execute(context.Background(), c.h)
	}
	c.pt.Run()
}

func (c *Console) ProcessTurn() {
	msg, err := c.agent.GetConversation().ProcessTurn(context.Background())
	if err != nil {
		fmt.Printf("Error processing turn: %v\n", err)
		return
	}
	for _, m := range msg {
		l := len(m.Content[0].String())
		if l > 128 {
			l = 128
		}
		// fmt.Println(m.Role, m.Content[0].String()[:l])
	}
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

func (c *Console) getFileSuggestions(pattern string) []prompt.Suggest {
	var suggestions []prompt.Suggest
	matches, err := filepath.Glob(pattern + "*")
	if err != nil {
		return suggestions
	}

	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		description := "file"
		if info.IsDir() {
			description = "directory"
		}
		suggestions = append(suggestions, prompt.Suggest{
			Text:        match,
			Description: description,
		})
	}
	return suggestions
}

func (c *Console) getFileManagerSuggestions(pattern string) []prompt.Suggest {
	filesInCtx := c.agent.GetConversation().GetCtx().GetFM().List(0)
	pattern = strings.ToLower(pattern)

	suggestions := make([]prompt.Suggest, 0)
	seen := make(map[string]bool)

	for filename := range filesInCtx {
		basename := filepath.Base(filename)
		lowerBasename := strings.ToLower(basename)

		// Skip if we've already seen this basename
		if seen[basename] {
			continue
		}
		seen[basename] = true

		// Check if pattern matches anywhere in the basename
		if strings.Contains(lowerBasename, pattern) {
			suggestions = append(suggestions, prompt.Suggest{
				Text:        basename,
				Description: filename,
			})
		}
	}

	// Sort suggestions alphabetically
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Text < suggestions[j].Text
	})

	return suggestions
}

func (c *Console) completer(d prompt.Document) []prompt.Suggest {
	var suggestions []prompt.Suggest
	input := d.TextBeforeCursor()
	words := strings.Fields(input)

	if len(words) <= 1 {
		// If the word starts with /, suggest commands
		word := d.GetWordBeforeCursor()
		if strings.HasPrefix(word, "/") {
			for cmdName, cmd := range c.commands {
				suggestions = append(suggestions, prompt.Suggest{
					Text:        cmdName,
					Description: cmd.description,
				})
			}
			return prompt.FilterHasPrefix(suggestions, word, true)
		}
	} else if words[0] == "/add" {
		// Get the word being typed
		word := d.GetWordBeforeCursor()
		// If word is empty, suggest current directory
		if word == "" {
			word = "."
		}
		return c.getFileSuggestions(word)
	} else if words[0] == "/remove" {
		// Get the word being typed
		word := d.GetWordBeforeCursor()
		// If word is empty, suggest current directory
		if word == "" {
			word = "."
		}
		return c.getFileManagerSuggestions(word)
	}

	return suggestions
}

func (c *Console) handleAddFile(args []string) {
	if len(args) == 0 {
		fmt.Println("Please specify file(s) to add")
		return
	}

	c.setState([]string{"add_files"})
	for _, file := range args {
		if err := c.agent.GetConversation().GetCtx().GetFM().Add(file, false); err != nil {
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

	for _, file := range args {
		c.agent.GetConversation().GetCtx().GetFM().Remove(file)
		fmt.Printf("Removed file: %s\n", file)
	}
}

func (c *Console) handleListFiles(args []string) {
	files := c.agent.GetConversation().GetCtx().GetFM().List(0)
	if len(files) == 0 {
		fmt.Println("No files currently attached")
		return
	}
	fmt.Println("Currently attached files:")
	for fileName, file := range files {
		fmt.Printf("- %s %s (%t)\n", fileName, file.LastUpdate, file.ReadOnly)
	}
}

func (c *Console) setState(args []string) {
	state := args[0]

	if state != c.agent.GetConversation().GetCurrentState() {
		c.ProcessTurn()
		err := c.agent.GetConversation().UpdateState(state)
		if err != nil {
			fmt.Printf("Error setting state: %v\n", err)
		}
	}
}

func (c *Console) send(args []string) {
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

func (c *Console) executor(input string) {
	input = strings.TrimSpace(input)

	if input == "" {
		return
	}

	// Save to history
	c.appendHistory(input)

	// Handle commands
	if strings.HasPrefix(input, "/") {
		parts := strings.Fields(input)
		slashCmd := parts[0]

		currentCmd := c.agent.GetConversation().GetCtx().GetCommand()
		scDef, found := currentCmd.SlashCommands[slashCmd]
		if found {
			// 2) If there's a next_state, do the transition
			newState := scDef.NextState
			if newState != "" {
				err := c.agent.GetConversation().UpdateState(newState)
				if err != nil {
					fmt.Printf("Error setting state: %v\n", err)
					return
				}
			}
			// 3) Pass leftover arguments to Input
			if len(parts) > 1 {
				argsOnly := strings.Join(parts[1:], " ")
				c.agent.GetConversation().GetCtx().SetVariable("Input", argsOnly)
			}
			fmt.Println("Command executed")
			// 4) Kick off the turn so the new template/handlers do the real logic
			c.ProcessTurn()
			return
		}

		cmd, exists := c.commands[parts[0]]
		if exists {
			cmd.handler(parts[1:])
			return
		}
		fmt.Printf("Unknown command: %s\n", parts[0])
		return
	}

	c.setState([]string{"user_input_code"})
	ctxMng := c.agent.GetConversation().GetCtx()
	// Add the command to the agent's message queue
	if input != "" {
		ctxMng.SetVariable("Input", input)
	}

	c.ProcessTurn()
}
