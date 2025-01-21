package prompts

import (
	"fmt"
	"testing"
)

func TestLoad(t *testing.T) {
	config := NewDefaultPromptsConfig()
	p := config.ArchitectPrompts
	data := map[string]interface{}{
		"Hash":     "abc123",
		"Message":  "Implement new feature",
		"Language": "English",
	}

	out, err := RenderTemplate(p.MainSystem, data)
	if err != nil {
		panic(err)
	}
	fmt.Println(out)
}
