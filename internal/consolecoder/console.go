package consolecoder

import (
	"fmt"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/y0ug/ai-helper/internal/assistant"
	"github.com/y0ug/ai-helper/pkg/highlighter"
)

type Console struct {
	coder       *assistant.BaseCoder
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
	return filepath.Join(usr.HomeDir, ".ai-coder-history")
}

func New(coder *assistant.BaseCoder, h *highlighter.Highlighter) *Console {
	c := &Console{
		coder:       coder,
		h:           h,
		historyFile: getHistoryFilePath(),
	}

	c.setCommands()

	c.pt = prompt.New(
		c.executor,
		c.completer,
		prompt.OptionLivePrefix(c.UpdatePrompt),
		prompt.OptionTitle("Chat"),
		prompt.OptionPrefix(fmt.Sprintf(" ➜ ")),
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
		"➜ "), false
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

func (c *Console) handleAddFile(args []string) {
	if len(args) == 0 {
		fmt.Println("Please specify file(s) to add")
		return
	}

	for _, file := range args {
		err := c.coder.GetRM().GetFM().Add(file, false)
		if err != nil {
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
		fmt.Printf("Removed file: %s\n", file)
		err := c.coder.GetRM().GetFM().Remove(file)
		if err != nil {
			fmt.Printf("Error removing file %s: %v\n", file, err)
		}
	}
}

func (c *Console) handleListFiles(args []string) {
	files := c.coder.GetRM().GetFM().List(0)
	if len(files) == 0 {
		fmt.Println("No files currently attached")
		return
	}
	fmt.Println("Currently attached files:")
	for fileName, file := range files {
		fmt.Printf(
			"- %s %s (%t) %s\n",
			fileName,
			file.LastUpdate,
			file.ReadOnly,
			string(file.Status.String()),
		)
	}
}

func (c *Console) send(args []string) {
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

		cmd, exists := c.commands[parts[0]]
		if exists {
			cmd.handler(parts[1:])
			return
		}
		fmt.Printf("Unknown command: %s\n", parts[0])
		return
	}

	// process input
	err := c.coder.Run(input)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
