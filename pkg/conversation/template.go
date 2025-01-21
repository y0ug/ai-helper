package conversation

import (
	"github.com/y0ug/ai-helper/internal/config"
)

// Template defines a template for a conversation state
type Template struct {
	ID           string
	Messages     []config.Message
	Variables    []config.Variable
	RequiredVars []string
	NextStates   []string
	Handlers     map[string]TurnHandler
	PreTurnCmds  []string
	PostTurnCmds []string
}

// NewTemplate creates a new template with initialized maps
func NewTemplate(id string) *Template {
	return &Template{
		ID:           id,
		Handlers:     make(map[string]TurnHandler),
		Messages:     make([]config.Message, 0),
		Variables:    make([]config.Variable, 0),
		RequiredVars: make([]string, 0),
		NextStates:   make([]string, 0),
		PreTurnCmds:  make([]string, 0),
		PostTurnCmds: make([]string, 0),
	}
}

func NewTemplatesFromConfig(tmpls map[string]config.Template) map[string]*Template {
	templates := make(map[string]*Template, 0)
	for id, tmpl := range tmpls {
		template := &Template{
			ID:           id,
			Messages:     tmpl.Messages,
			Variables:    tmpl.Variables,
			RequiredVars: extractRequiredVars(tmpl.Variables),
			NextStates:   tmpl.NextStates,
			Handlers:     make(map[string]TurnHandler),
			PreTurnCmds:  tmpl.PreTurnCmds,
			PostTurnCmds: tmpl.PostTurnCmds,
		}

		// Register handlers
		for _, handlerName := range tmpl.Handlers {
			switch handlerName {
			case "codediff":
				template.Handlers[handlerName] = NewCodeDiffHandler()
			case "add_file":
				template.Handlers[handlerName] = NewAddFileHandler()
			}
		}
		templates[id] = template
	}
	return templates
}

// extractRequiredVars extracts required variable names from Variable slice
func extractRequiredVars(vars []config.Variable) []string {
	required := make([]string, 0)
	for _, v := range vars {
		if v.Type != "" {
			required = append(required, v.Name)
		}
	}
	return required
}
