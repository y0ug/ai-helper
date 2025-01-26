package consolecoder

// Command represents a chat command
type Command struct {
	name        string
	description string
	handler     func(args []string)
}

func (c *Console) setCommands() {
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
		"/send": {
			name:        "send",
			description: "send request",
			handler:     c.send,
		},
	}
}

func (c *Console) handleQuit(args []string) {
	c.shutdown()
}
