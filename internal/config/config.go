package config

import (
	"fmt"
	"os/exec"
	"strings"
)

const (
	DefaultConfigFile = "ai-helper.yaml"
)

// GetConfig loads and returns the configuration
func GetConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = DefaultConfigFile
	}

	loader := NewLoader()
	return loader.Load(configPath)
}

// GetCommandPrompt returns the prompt and system prompt for a given command name
func (c *Config) GetCommandPrompt(name string) (string, string, error) {
	cmd, exists := c.Commands[name]
	if !exists {
		return "", "", fmt.Errorf("command '%s' not found in configuration", name)
	}
	return cmd.Prompt, cmd.System, nil
}

// ValidateConfig checks if the configuration is valid
func (c *Config) ValidateConfig() error {
	if len(c.Commands) == 0 {
		return fmt.Errorf("no commands defined in configuration")
	}

	for name, cmd := range c.Commands {
		if cmd.Prompt == "" {
			return fmt.Errorf("empty prompt for command '%s'", name)
		}
	}

	return nil
}

// LoadPromptContent loads the prompt content, system prompt, and processes any variables
func LoadPromptContent(cmd Command) (string, string, map[string]interface{}, error) {
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
				return "", "", nil, fmt.Errorf("error executing command %s: %w", v.Exec, err)
			}
			vars[v.Name] = strings.TrimSpace(string(out))
		} else if v.Name != "" {
			// Store regular variable name for later use
			vars[v.Name] = ""
		}
	}

	return cmd.Prompt, cmd.System, vars, nil
}
