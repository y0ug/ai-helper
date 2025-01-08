package prompt

import (
	"bytes"
	"os"
	"path/filepath"
	"text/template"
)

// TemplateData holds all data available to templates
type TemplateData struct {
	Input  string
	Env    map[string]string
	Files  map[string]string
	Vars   map[string]interface{}
}

// NewTemplateData creates a new TemplateData with initialized maps
func NewTemplateData(input string) *TemplateData {
	return &TemplateData{
		Input: input,
		Env:   make(map[string]string),
		Files: make(map[string]string),
		Vars:  make(map[string]interface{}),
	}
}

// LoadEnvironment loads all environment variables into the template data
func (td *TemplateData) LoadEnvironment() {
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			td.Env[pair[0]] = pair[1]
		}
	}
}

// LoadFiles loads content of specified files into the template data
func (td *TemplateData) LoadFiles(paths []string) error {
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading file %s: %w", path, err)
		}
		td.Files[path] = string(content)
	}
	return nil
}

// GetTemplateFuncs returns the map of template helper functions
func GetTemplateFuncs(td *TemplateData) template.FuncMap {
	return template.FuncMap{
		"fileContent": func(path string) string {
			content, ok := td.Files[path]
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
			content, ok := td.Files[path]
			if !ok {
				return fmt.Sprintf("Error: file %s not found", path)
			}
			ext := filepath.Ext(path)
			return fmt.Sprintf("```%s\n%s\n```", ext[1:], content)
		},
	}
}

// Execute processes a template with the provided template data
func Execute(templateContent string, data *TemplateData) (string, error) {
	tmpl, err := template.New("prompt").
		Funcs(GetTemplateFuncs(data)).
		Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return buf.String(), nil
}
