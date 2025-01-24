package executors

import (
	"fmt"
	"os/exec"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

var Fence = "`"

func (c *ExecutorLocal) ShellCommandHandler(
	command string,
) ([]actions.Action[any], error) {
	results := make([]actions.Action[any], 0)

	c.logger.Info("running shell command", "command", command)

	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	commandOutput := string(output)

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	msg := fmt.Sprintf(
		"# Shell command result\n\nshell command:\n```\n%s\n```\n\nExit code: %d\n\n```\n%s\n```\n",
		command,
		exitCode,
		commandOutput,
	)

	c.logger.Info("shell command msg", "msg", msg)

	action := actions.NewParsedAction(actions.SendChatMessage{
		Msg:        *chat.NewUserMessage(msg),
		NeedRender: false,
	})

	results = append(results, action)
	return results, nil
}
