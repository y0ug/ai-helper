package conversation

// Template defines a template for a conversation state
type Template struct {
	ID           string
	SystemPrompt string
	UserPrompt   string
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
		RequiredVars: make([]string, 0),
		NextStates:   make([]string, 0),
		PreTurnCmds:  make([]string, 0),
		PostTurnCmds: make([]string, 0),
	}
}
