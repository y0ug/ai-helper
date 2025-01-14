package common

import (
	"encoding/json"
)

type WrapperStream[T any] struct {
	original Streamer[T]
	provider string
	current  LLMStreamEvent
	err      error
}

func NewWrapperStream[T any](
	original Streamer[T],
	provider string,
) Streamer[LLMStreamEvent] {
	return &WrapperStream[T]{
		original: original,
		provider: provider,
	}
}

func (w *WrapperStream[T]) Next() bool {
	if w.err != nil {
		return false
	}

	if w.original.Next() {
		event := w.original.Current()
		data, err := json.Marshal(event)
		if err != nil {
			w.err = err
			return false
		}

		// Extract event type based on provider
		eventType := "unknown"
		switch w.provider {
		case "anthropic":
			// var ae anthropic.MessageStreamEvent
			// if err := json.Unmarshal(data, &ae); err == nil {
			// 	eventType = ae.Type
			// }
		case "openai":
			// var oe openai.ChatCompletionChunk
			// if err := json.Unmarshal(data, &oe); err == nil {
			// 	eventType = "chat_completion_chunk" // Adjust as needed
			// }

		}

		w.current = LLMStreamEvent{
			Provider: w.provider,
			Type:     eventType,
			Data:     data,
		}
		return true
	}

	w.err = w.original.Err()
	return false
}

func (w *WrapperStream[T]) Current() LLMStreamEvent {
	return w.current
}

func (w *WrapperStream[T]) Err() error {
	return w.err
}

func (w *WrapperStream[T]) Close() error {
	return w.original.Close()
}
