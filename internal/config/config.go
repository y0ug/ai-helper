package config

import (
	"fmt"
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

// GetCommandPrompt returns the prompt for a given command name
func (c *Config) GetCommandPrompt(name string) (string, error) {
	cmd, exists := c.Commands[name]
	if !exists {
		return "", fmt.Errorf("command '%s' not found in configuration", name)
	}
	return cmd.Prompt, nil
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
