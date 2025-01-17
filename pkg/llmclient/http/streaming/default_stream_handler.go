package streaming

import (
	"encoding/json"
	"fmt"
)

// DefaultStreamHandler implements StreamHandler for basic streaming responses
type DefaultStreamHandler[T any, TypeIn Event] struct{}

func NewDefaultStreamHandler[T any, TypeIn Event]() *DefaultStreamHandler[T, Event] {
	return &DefaultStreamHandler[T, Event]{}
}

func (h *DefaultStreamHandler[T, TypeIn]) HandleEvent(event Event) (T, error) {
	var result T

	if len(event.Data) == 0 {
		return result, nil
	}

	if err := json.Unmarshal(event.Data, &result); err != nil {
		return result, fmt.Errorf("error unmarshalling event: %w %s", err, event.Data)
	}

	return result, nil
}

func (h *DefaultStreamHandler[T, TypeIn]) ShouldContinue(event Event) bool {
	// fmt.Printf("[DONE] != '%s'\n", string(event.Data))
	return string(event.Data) != "[DONE]"
}
