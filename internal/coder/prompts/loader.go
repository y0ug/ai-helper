package prompts

import (
	"bytes"
	"text/template"
)

// RenderTemplate applies Go's text/template to a prompt, e.g. filling in {{.Hash}} etc.
func RenderTemplate(prompt string, data interface{}) (string, error) {
	tmpl, err := template.New("prompt").Parse(prompt)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	return buf.String(), err
}
