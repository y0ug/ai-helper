package config

import (
	"strings"
)

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

func (c *Template) NeedInput() bool {
	for _, v := range c.Variables {
		if v.Name == "Input" {
			return true
		}
	}
	return false
}

// We don't want this since we will have to process the template
// func (m *Message) ToChatMessage() chat.ChatMessage {
// 	return *chat.NewMessage(m.Role, chat.NewTextContent(m.Content))
// }

// Message represents a chat message with role and content
type Message struct {
	Role    string `yaml:"role"    json:"role"`
	Content string `yaml:"content" json:"content"`
}

// Template represents a single conversation state template
type Template struct {
	Description  string     `yaml:"description,omitempty"    json:"description,omitempty"`
	Messages     []Message  `yaml:"messages,omitempty"       json:"messages,omitempty"`
	Variables    []Variable `yaml:"variables,omitempty"      json:"variables,omitempty"`
	NextStates   []string   `yaml:"next_states,omitempty"    json:"next_states,omitempty"`
	Handlers     []string   `yaml:"handlers,omitempty"       json:"handlers,omitempty"`
	PreTurnCmds  []string   `yaml:"pre_turn_cmds,omitempty"  json:"pre_turn_cmds,omitempty"`
	PostTurnCmds []string   `yaml:"post_turn_cmds,omitempty" json:"post_turn_cmds,omitempty"`
}

// Command represents a single AI command configuration
type Command struct {
	Name         string              `yaml:"-"                       json:"-"`
	Description  string              `yaml:"description,omitempty"   json:"description,omitempty"`
	Templates    map[string]Template `yaml:"templates"               json:"templates"`
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
