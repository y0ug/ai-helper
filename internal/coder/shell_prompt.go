// pkg/coders/shell_prompt.go
package coder

import (
	"bytes"
	"os/exec"
	"runtime"

	promptsA "github.com/y0ug/ai-helper/internal/coder/prompts"
)

func (c *BaseCoder) getShellCmdPrompt() string {
	// If shell commands are disabled, return no shell command prompt
	if !c.suggestShellCommands {
		prompts := c.getPrompts()
		return prompts.NoShellCmdPrompt
	}

	// Get base prompt
	prompts := c.getPrompts()
	shellPrompt := prompts.ShellCmdPrompt

	// Format with platform info
	data := TemplateData{
		Platform: c.getPlatformInfo(),
	}
	formatted, err := promptsA.RenderTemplate(shellPrompt, data)
	if err != nil {
		return shellPrompt
	}

	return formatted
}

func (c *BaseCoder) getShellCmdReminder() string {
	if !c.suggestShellCommands {
		prompts := c.getPrompts()
		return prompts.NoShellCmdReminder
	}

	prompts := c.getPrompts()
	reminder := prompts.ShellCmdReminder

	data := TemplateData{
		Platform: c.getPlatformInfo(),
	}
	formatted, err := promptsA.RenderTemplate(reminder, data)
	if err != nil {
		return reminder
	}

	return formatted
}

// handleShellCommands processes suggested shell commands
type ConfirmGroup struct{}

func (c *BaseCoder) handleShellCommands(commandStr string, group ConfirmGroup) string {
	// // Split commands into individual lines
	// commands := strings.Split(strings.TrimSpace(commandStr), "\n")
	//
	// // Count actual commands (skip comments and empty lines)
	// commandCount := 0
	// for _, cmd := range commands {
	// 	cmd = strings.TrimSpace(cmd)
	// 	if cmd != "" && !strings.HasPrefix(cmd, "#") {
	// 		commandCount++
	// 	}
	// }
	//
	// // Build prompt based on number of commands
	// prompt := "Run shell command?"
	// if commandCount > 1 {
	// 	prompt = "Run shell commands?"
	// }
	//
	// // Ask for confirmation
	// if !c.io.ConfirmAsk(prompt, strings.Join(commands, "\n"), true, group, true) {
	// 	return ""
	// }
	//
	// var output strings.Builder
	// for _, cmd := range commands {
	// 	cmd = strings.TrimSpace(cmd)
	// 	if cmd == "" || strings.HasPrefix(cmd, "#") {
	// 		continue
	// 	}
	//
	// 	c.io.ToolOutput("")
	// 	c.io.ToolOutput("Running " + cmd)
	//
	// 	// Add command to input history
	// 	c.io.AddToInputHistory("/run " + cmd)
	//
	// 	// Run command
	// 	exitStatus, cmdOutput := RunCmd(cmd, c.io.ToolError, c.root)
	// 	if cmdOutput != "" {
	// 		output.WriteString("Output from " + cmd + "\n")
	// 		output.WriteString(cmdOutput + "\n")
	// 	}
	// }
	//
	// // If there's output, ask to add it to chat
	// finalOutput := output.String()
	// if strings.TrimSpace(finalOutput) != "" {
	// 	if c.io.ConfirmAsk("Add command output to the chat?", "", false, nil, true) {
	// 		numLines := len(strings.Split(strings.TrimSpace(finalOutput), "\n"))
	// 		linePlural := "line"
	// 		if numLines != 1 {
	// 			linePlural = "lines"
	// 		}
	// 		c.io.ToolOutput(fmt.Sprintf("Added %d %s of output to the chat.", numLines, linePlural))
	// 		return finalOutput
	// 	}
	// }

	return ""
}

// RunCmd executes a shell command and returns its output
func RunCmd(cmd string, errPrinter func(string), workDir string) (int, string) {
	var shell, flag string
	if runtime.GOOS == "windows" {
		shell = "cmd"
		flag = "/C"
	} else {
		shell = "sh"
		flag = "-c"
	}

	command := exec.Command(shell, flag, cmd)
	command.Dir = workDir

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	exitStatus := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitStatus = exitErr.ExitCode()
		}
	}

	if stderr.Len() > 0 {
		errPrinter(stderr.String())
	}

	return exitStatus, stdout.String()
}
