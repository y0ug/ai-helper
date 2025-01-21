package context

import (
	"html/template"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/filemanager"
)

// ContextManager handles all context-related operations for a conversation
type ContextManager interface {
	// Template operations
	ExecuteTemplate(templateText string) (string, error)

	// Variable management
	SetVariable(key string, value interface{})
	GetVariable(key string) (interface{}, bool)
	LoadVariables(vars map[string]interface{}) error

	// File operations
	GetFM() filemanager.FileManager

	// Environment
	LoadEnvironment() error
	GetEnv(key string) (string, bool)

	// Command/Config access
	GetCommand() *config.Command

	// Template functions
	GetTemplateFuncs() template.FuncMap
}
