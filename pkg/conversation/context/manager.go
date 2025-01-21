package context

import (
	"bytes"
	"fmt"
	"html/template"
	"maps"
	"os"
	"strings"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/filemanager"
)

// conversationContext implements ContextManager
type conversationContext struct {
	cmd  *config.Command
	env  map[string]string
	fm   filemanager.FileManager
	vars map[string]interface{}
}

func NewContext(cmd *config.Command, fm filemanager.FileManager) *conversationContext {
	return &conversationContext{
		cmd:  cmd,
		env:  make(map[string]string),
		fm:   fm,
		vars: make(map[string]interface{}),
	}
}

// ExecuteTemplate processes a template with the provided data
func (cc *conversationContext) ExecuteTemplate(
	templateID string,
	templateText string,
) (string, error) {
	tmpl, err := template.New(templateID).
		Funcs(cc.GetTemplateFuncs()).
		Parse(templateText)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	varProcessor := NewVariableProcessor(cc.cmd)
	vars, err := varProcessor.Process(templateID, cc.vars)
	varsCopy := cc.vars
	if err != nil {
		err = fmt.Errorf("error processing variables: %w", err)
		fmt.Println(err)
	} else {
		maps.Copy(varsCopy, vars)
	}
	data := TemplateData{
		Env:   cc.env,
		Files: cc.fm.GetFiles(),
		Vars:  varsCopy,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
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
	return cc.cmd
}

// GetTemplateFuncs returns template helper functions
func (cc *conversationContext) GetTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"DumpNewFiles": func() string {
			files := cc.GetFM().GetNewFiles()

			var sbReadOnly, sb strings.Builder
			// sbReadOnly.WriteString(`Do not propose changes to these files, treat them as *read-only*
			// If you need to edit any of these files, ask me to *add them to the chat* first.\n\n`)

			// sb.WriteString(`You can propose changes to these files.\n\n`)
			for path, file := range files {
				if file.ReadOnly {
					sbReadOnly.WriteString(fmt.Sprintf("\n%s\n```\n%s\n```\n", path, file.Content))
				} else {
					sb.WriteString(fmt.Sprintf("\n%s\n```\n%s\n```\n", path, file.Content))
				}
			}

			// sbReadOnly.WriteString("\n\n---\n\n")
			// sbReadOnly.WriteString(sb.String())
			// sbReadOnly.WriteString("\n\n---\n\n")
			return sb.String()
		},
	}
}
