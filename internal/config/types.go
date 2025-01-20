package config

import "strings"

// Variable types
const (
	VarTypeExec  = "exec"
	VarTypeArg   = "arg"
	VarTypeStdin = "stdin"
)

// Variable represents a variable definition in command configuration
type Variable struct {
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	Type string `yaml:"type,omitempty" json:"type,omitempty"` // Can be combination like "stdin|arg|exec"
	Exec string `yaml:"exec,omitempty" json:"exec,omitempty"`
}

// GetTypes returns the variable types as a slice
func (v *Variable) GetTypes() []string {
	if v.Type == "" {
		return nil
	}
	return strings.Split(v.Type, "|")
}

// Template represents a single conversation state template
type Template struct {
	Description  string     `yaml:"description,omitempty"   json:"description,omitempty"`
	System       string     `yaml:"system,omitempty"        json:"system,omitempty"`
	Prompt       string     `yaml:"prompt"                  json:"prompt"`
	Variables    []Variable `yaml:"variables,omitempty"     json:"variables,omitempty"`
	NextStates   []string   `yaml:"next_states,omitempty"   json:"next_states,omitempty"`
	Handlers     []string   `yaml:"handlers,omitempty"      json:"handlers,omitempty"`
}

// Command represents a single AI command configuration
type Command struct {
	Description  string              `yaml:"description,omitempty"   json:"description,omitempty"`
	Templates    map[string]Template `yaml:"templates"               json:"templates"`
	Input        bool                `yaml:"input,omitempty"         json:"input,omitempty"`
	InputCommand string              `yaml:"input_command,omitempty" json:"input_command,omitempty"`
	Files        []string            `yaml:"files,omitempty"         json:"files,omitempty"`
	MCPServers   []string            `yaml:"mcpServers,omitempty"    json:"mcpServers,omitempty"`
	InitialState string              `yaml:"initial_state,omitempty" json:"initial_state,omitempty"`
}

// MCPServer represents a single MCP server configuration
type MCPServer struct {
	Command string   `yaml:"command" json:"command"`
	Args    []string `yaml:"args"    json:"args"`
}

// MCPServers represents a map of MCP server configurations
type MCPServers map[string]MCPServer

// Config represents the root configuration structure
type Config struct {
	Commands   map[string]Command `yaml:"commands"             json:"commands"`
	MCPServers MCPServers         `yaml:"mcpServers,omitempty" json:"mcpServers,omitempty"`
}
