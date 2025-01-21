package context

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/filemanager"
)

// conversationContext implements ContextManager
type conversationContext struct {
	command *config.Command
	env     map[string]string
	fm      filemanager.FileManager
	vars    map[string]interface{}
}

func NewContext(cmd *config.Command, fm filemanager.FileManager) *conversationContext {
	return &conversationContext{
		command: cmd,
		env:     make(map[string]string),
		fm:      fm,
		vars:    make(map[string]interface{}),
	}
}

// ExecuteTemplate processes a template with the provided data
func (cc *conversationContext) ExecuteTemplate(templateText string) (string, error) {
	tmpl, err := template.New("prompt").
		Funcs(cc.GetTemplateFuncs()).
		Parse(templateText)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cc); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return buf.String(), nil
}

// SetVariable sets a variable in the context
func (cc *conversationContext) SetVariable(key string, value interface{}) {
	cc.vars[key] = value
}

// GetVariable gets a variable from the context
func (cc *conversationContext) GetVariable(key string) (interface{}, bool) {
	value, exists := cc.vars[key]
	return value, exists
}

// LoadVariables loads multiple variables into the context
func (cc *conversationContext) LoadVariables(vars map[string]interface{}) error {
	for k, v := range vars {
		cc.vars[k] = v
	}
	return nil
}

// GetFM returns the FileManager instance
func (cc *conversationContext) GetFM() filemanager.FileManager {
	return cc.fm
}

// LoadEnvironment loads all environment variables into the context
func (cc *conversationContext) LoadEnvironment() error {
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			cc.env[pair[0]] = pair[1]
		}
	}
	return nil
}

// GetEnv retrieves an environment variable
func (cc *conversationContext) GetEnv(key string) (string, bool) {
	value, exists := cc.env[key]
	return value, exists
}

// GetCommand returns the current command configuration
func (cc *conversationContext) GetCommand() *config.Command {
	return cc.command
}

// GetTemplateFuncs returns template helper functions
func (cc *conversationContext) GetTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"fileContent": func(path string) string {
			content, err := cc.fm.ReadFile(path)
			if err != nil {
				return fmt.Sprintf("Error: file %s not found", path)
			}
			return string(content)
		},
		"fileExt": filepath.Ext,
		"fileName": func(path string) string {
			return filepath.Base(path)
		},
		"formatFile": func(path string) string {
			content, err := cc.fm.ReadFile(path)
			if err != nil {
				return fmt.Sprintf("Error: file %s not found", path)
			}
			ext := filepath.Ext(path)
			return fmt.Sprintf("```%s\n%s\n```", ext[1:], string(content))
		},
	}
}
