package conversation

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/y0ug/ai-helper/internal/config"
)

// RequestContext holds all data available to templates
type RequestContext struct {
	cmd   *config.Command
	Env   map[string]string
	Files map[string]string
	Vars  map[string]interface{}
}

// NewRequestContext creates a new RequestContext with initialized maps
func NewRequestContext(cmd *config.Command) *RequestContext {
	req := &RequestContext{
		cmd:   cmd,
		Env:   make(map[string]string),
		Files: make(map[string]string),
		Vars:  make(map[string]interface{}),
	}

	req.LoadEnvironment()
	for _, file := range cmd.Files {
		req.LoadFiles(file)
	}
	return req
}

// LoadEnvironment loads all environment variables into the template data
func (rc *RequestContext) LoadEnvironment() {
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			rc.Env[pair[0]] = pair[1]
		}
	}
}

// LoadFiles loads content of specified files into the template data
func (rc *RequestContext) LoadFiles(paths ...string) error {
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading file %s: %w", path, err)
		}
		rc.Files[path] = string(content)
	}
	return nil
}

// Execute processes a template with the provided data
func (rc *RequestContext) Execute(templateText string) (string, error) {
	tmpl, err := template.New("prompt").
		Funcs(rc.GetTemplateFuncs()).
		Parse(templateText)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, rc); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return buf.String(), nil
}

// Process handles variable processing and resolution
func (rc *RequestContext) Process(args map[string]string) error {
	var variables []config.Variable
	if rc.cmd.InitialState != "" {
		if tmpl, ok := rc.cmd.Templates[rc.cmd.InitialState]; ok {
			variables = tmpl.Variables
		}
	} else {
		for _, tmpl := range rc.cmd.Templates {
			variables = tmpl.Variables
			break
		}
	}

	for _, v := range variables {
		types := v.GetTypes()
		if len(types) == 0 {
			continue
		}

		var value string
		for _, t := range types {
			switch t {
			case config.VarTypeExec:
				if v.Exec != "" {
					cmd := exec.Command("sh", "-c", v.Exec)
					output, err := cmd.Output()
					if err == nil {
						value = strings.TrimSpace(string(output))
						break
					}
				}
			case config.VarTypeArg:
				if val, ok := args[v.Name]; ok {
					value = val
					break
				}
			case config.VarTypeStdin:
				val, err := readStdin()
				if err == nil {
					value = val
					break
				}
			}

			if value != "" {
				break
			}
		}

		if value == "" {
			return fmt.Errorf("no value found for variable %s after trying types: %v", v.Name, types)
		}

		rc.Vars[v.Name] = value
	}
	return nil
}

// GetTemplateFuncs returns template helper functions
func (rc *RequestContext) GetTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"fileContent": func(path string) string {
			content, ok := rc.Files[path]
			if !ok {
				return fmt.Sprintf("Error: file %s not found", path)
			}
			return content
		},
		"fileExt": filepath.Ext,
		"fileName": func(path string) string {
			return filepath.Base(path)
		},
		"formatFile": func(path string) string {
			content, ok := rc.Files[path]
			if !ok {
				return fmt.Sprintf("Error: file %s not found", path)
			}
			ext := filepath.Ext(path)
			return fmt.Sprintf("```%s\n%s\n```", ext[1:], content)
		},
		"FilesDump": rc.FormatFiles,
	}
}

// FormatFiles formats all files in the context
func (rc *RequestContext) FormatFiles() string {
	var sb strings.Builder
	for path, content := range rc.Files {
		sb.WriteString(fmt.Sprintf("\n%s\n```\n%s\n```\n", path, content))
	}
	return sb.String()
}

// SetVariable sets a variable in the context
func (rc *RequestContext) SetVariable(key string, value interface{}) {
	rc.Vars[key] = value
}

// GetVariable gets a variable from the context
func (rc *RequestContext) GetVariable(key string) (interface{}, bool) {
	value, exists := rc.Vars[key]
	return value, exists
}

// readStdin reads input from stdin if available
func readStdin() (string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat stdin: %w", err)
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		reader := bufio.NewReader(os.Stdin)
		var builder strings.Builder

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				return "", fmt.Errorf("failed to read stdin: %w", err)
			}
			builder.WriteString(line)
		}

		if builder.Len() > 0 {
			return strings.TrimSpace(builder.String()), nil
		}
	}
	return "", fmt.Errorf("no input provided")
}
