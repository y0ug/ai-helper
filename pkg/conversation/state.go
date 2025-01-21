package conversation

import (
	"fmt"
	"strings"

	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
)

// DefaultStateTransitioner provides basic state transition logic
type DefaultStateTransitioner struct{}

// NewDefaultStateTransitioner creates a new default state transitioner
func NewDefaultStateTransitioner() *DefaultStateTransitioner {
	return &DefaultStateTransitioner{}
}

// DetermineNextState implements the StateTransitioner interface
func (t *DefaultStateTransitioner) DetermineNextState(
	cm Manager,
	responses []*chat.ChatResponse,
) (string, error) {
	if len(responses) == 0 {
		return "", fmt.Errorf("no responses available for state transition")
	}

	template := cm.GetCurrentTemplate()

	// Get the last response content
	lastResponse := responses[len(responses)-1]
	content := lastResponse.Choice[0].Content[0].String()

	// Check if there's an explicit state transition command
	if strings.HasPrefix(strings.ToLower(content), "!state ") {
		requestedState := strings.TrimPrefix(strings.ToLower(content), "!state ")
		requestedState = strings.TrimSpace(requestedState)

		// Verify the requested state is valid
		for _, validState := range template.NextStates {
			if strings.ToLower(validState) == requestedState {
				return validState, nil
			}
		}
		return "", fmt.Errorf("invalid state transition requested: %s", requestedState)
	}

	// If no explicit transition, stay in current state if it's in NextStates
	for _, state := range template.NextStates {
		if state == template.ID {
			return template.ID, nil
		}
	}

	// If current state isn't in NextStates, use the first available next state
	if len(template.NextStates) > 0 {
		return template.NextStates[0], nil
	}

	return "", fmt.Errorf("no valid next state available")
}
