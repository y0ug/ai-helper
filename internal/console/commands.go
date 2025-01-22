package console

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
		"/state": {
			name:        "state",
			description: "Set state",
			handler:     c.setState,
		},
		"/send": {
			name:        "send",
			description: "send request",
			handler:     c.send,
		},
	}
}
