package config

import (
	"fmt"
	"os/exec"
	"strings"
)

// LoadPromptContent loads the prompt content and processes any variables
func LoadPromptContent(cmd Command) (string, map[string]interface{}, error) {
	vars := make(map[string]interface{})

	// Process any variables defined in the command
	for _, v := range cmd.Variables {
		if v.Name == "Input" {
			continue
		}
		if v.Type == "exec" && v.Exec != "" && v.Name != "" {
			// Execute command and capture output
			out, err := exec.Command("sh", "-c", v.Exec).Output()
			if err != nil {
				return "", nil, fmt.Errorf("error executing command %s: %w", v.Exec, err)
			}
			vars[v.Name] = strings.TrimSpace(string(out))
		} else if v.Name != "" {
			// Store regular variable name for later use
			vars[v.Name] = ""
		}
	}

	return cmd.Prompt, vars, nil
}
