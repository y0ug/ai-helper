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

// ValidateConfig checks if the configuration is valid
func (c *Config) ValidateConfig() error {
	for name, cmd := range c.Commands {
		if len(cmd.Templates) == 0 {
			return fmt.Errorf("no templates defined for command '%s'", name)
		}

		// Validate templates
		for tname, tmpl := range cmd.Templates {
			if len(tmpl.Messages) == 0 {
				return fmt.Errorf(
					"template '%s' in command '%s' must have either prompt or messages defined",
					tname,
					name,
				)
			}
		}

		// Validate that referenced MCP servers exist
		for _, serverName := range cmd.MCPServers {
			if _, exists := c.MCPServers[serverName]; !exists {
				return fmt.Errorf(
					"command '%s' references non-existent MCP server '%s'",
					name,
					serverName,
				)
			}
		}
	}

	// Validate MCP server configurations if present
	for name, server := range c.MCPServers {
		if server.Command == "" {
			return fmt.Errorf("empty command for MCP server '%s'", name)
		}
	}

	return nil
}
