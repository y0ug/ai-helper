package ssestream

import (
	"encoding/json"
	"fmt"
)

// DefaultStreamHandler implements StreamHandler for basic streaming responses
type DefaultStreamHandler[T any] struct{}

func NewDefaultStreamHandler[T any]() *DefaultStreamHandler[T] {
	return &DefaultStreamHandler[T]{}
}

func (h *DefaultStreamHandler[T]) HandleEvent(event Event) (T, error) {
	var result T

	if len(event.Data) == 0 {
		return result, nil
	}

	if err := json.Unmarshal(event.Data, &result); err != nil {
		return result, fmt.Errorf("error unmarshalling event: %w", err)
	}

	return result, nil
}

func (h *DefaultStreamHandler[T]) ShouldContinue(event Event) bool {
	return string(event.Data) != "[DONE]"
}
