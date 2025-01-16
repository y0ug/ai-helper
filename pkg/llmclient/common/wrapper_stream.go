package common

import (
	"encoding/json"
)

type WrapperStream[T any] struct {
	original Streamer[T]
	handler  ProviderEventHandler
	current  StreamEvent
	err      error
}

func NewWrapperStream[T any](
	original Streamer[T],
	handler ProviderEventHandler,
) Streamer[StreamEvent] {
	return &WrapperStream[T]{
		original: original,
		handler:  handler,
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

		w.current = w.handler.ProcessEvent(data)
		return true
	}

	w.err = w.original.Err()
	return false
}

func (w *WrapperStream[T]) Current() StreamEvent {
	return w.current
}

func (w *WrapperStream[T]) Err() error {
	return w.err
}

func (w *WrapperStream[T]) Close() error {
	return w.original.Close()
}
