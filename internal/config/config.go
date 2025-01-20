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

// GetCommandPrompt returns the prompt and system prompt for a given command name and template
func (c *Config) GetCommandPrompt(name string) (string, string, error) {
	cmd, exists := c.Commands[name]
	if !exists {
		return "", "", fmt.Errorf("command '%s' not found in configuration", name)
	}

	// Get initial template or first available template
	var tmpl Template
	if cmd.InitialState != "" {
		if t, ok := cmd.Templates[cmd.InitialState]; ok {
			tmpl = t
		}
	} else {
		// Get first template
		for _, t := range cmd.Templates {
			tmpl = t
			break
		}
	}

	return tmpl.Prompt, tmpl.System, nil
}

// ValidateConfig checks if the configuration is valid
func (c *Config) ValidateConfig() error {
	for name, cmd := range c.Commands {
		if len(cmd.Templates) == 0 {
			return fmt.Errorf("no templates defined for command '%s'", name)
		}

		// Validate templates
		for tname, tmpl := range cmd.Templates {
			if tmpl.Prompt == "" {
				return fmt.Errorf("empty prompt for template '%s' in command '%s'", tname, name)
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

// LoadPromptContent loads the prompt content, system prompt, and processes any variables
func LoadPromptContent(cmd Command) (string, string, map[string]interface{}, error) {
	vars := make(map[string]interface{})

	// Get initial or first template
	var tmpl Template
	if cmd.InitialState != "" {
		if t, ok := cmd.Templates[cmd.InitialState]; ok {
			tmpl = t
		}
	} else {
		// Get first template
		for _, t := range cmd.Templates {
			tmpl = t
			break
		}
	}

	// Process any variables defined in the template
	for _, v := range tmpl.Variables {
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

	return tmpl.Prompt, tmpl.System, vars, nil
}
