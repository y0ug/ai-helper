package config

// Variable represents a variable definition in command configuration
type Variable struct {
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	Type string `yaml:"type,omitempty" json:"type,omitempty"`
	Exec string `yaml:"exec,omitempty" json:"exec,omitempty"`
}

// Command represents a single AI command configuration
type Command struct {
	Description  string     `yaml:"description,omitempty"   json:"description,omitempty"`
	System       string     `yaml:"system,omitempty"        json:"system,omitempty"`
	Prompt       string     `yaml:"prompt"                  json:"prompt"`
	Variables    []Variable `yaml:"variables,omitempty"     json:"variables,omitempty"`
	Input        bool       `yaml:"input,omitempty"         json:"input,omitempty"`
	InputCommand string     `yaml:"input_command,omitempty" json:"input_command,omitempty"`
	Files        []string   `yaml:"files,omitempty"         json:"files,omitempty"`
}

// MCPServer represents a single MCP server configuration
type MCPServer struct {
	Command string   `yaml:"command" json:"command"`
	Args    []string `yaml:"args" json:"args"`
}

// MCPServers represents a map of MCP server configurations
type MCPServers map[string]MCPServer

// Config represents the root configuration structure
type Config struct {
	Commands    map[string]Command `yaml:"commands" json:"commands"`
	MCPServers  MCPServers        `yaml:"mcpServers,omitempty" json:"mcpServers,omitempty"`
}
