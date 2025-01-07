package io

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// ReadInput reads input based on the specified input types
func ReadInput(args []string, inputTypes []string, fallbackCmd string) (string, error) {
	for _, inputType := range inputTypes {
		switch inputType {
		case "arg":
			if len(args) > 0 {
				return strings.Join(args, " "), nil
			}
		case "stdin":
			stat, err := os.Stdin.Stat()
			if err != nil {
				return "", fmt.Errorf("failed to stat stdin: %w", err)
			}

			if (stat.Mode() & os.ModeCharDevice) == 0 {
				reader := bufio.NewReader(os.Stdin)
				var builder strings.Builder

				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						if err == io.EOF {
							break
						}
						return "", fmt.Errorf("failed to read stdin: %w", err)
					}
					builder.WriteString(line)
				}

				if builder.Len() > 0 {
					return strings.TrimSpace(builder.String()), nil
				}
			}
		}
	}

	// If we reach here and have a fallback command, execute it
	if fallbackCmd != "" {
		cmd := exec.Command("sh", "-c", fallbackCmd)
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to execute fallback command: %w", err)
		}
		return strings.TrimSpace(string(output)), nil
	}

	return "", fmt.Errorf("no input provided through specified methods")
}
