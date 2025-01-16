package common

// ProviderEventStream adapts provider-specific streams to a normalized StreamEvent interface
type ProviderEventStream[T any] struct {
    original Streamer[T]
    handler  ProviderEventHandler[T]
    current  StreamEvent
    err      error
}

// NewProviderEventStream creates a new stream that normalizes provider events
func NewProviderEventStream[T any](
    original Streamer[T],
    handler ProviderEventHandler[T],
) Streamer[StreamEvent] {
    return &ProviderEventStream[T]{
        original: original,
        handler:  handler,
    }
}

func (w *ProviderEventStream[T]) Next() bool {
    if w.err != nil {
        return false
    }

    if w.original.Next() {
        event := w.original.Current()
        w.current = w.handler.ProcessEvent(event)
        return true
    }

    w.err = w.original.Err()
    return false
}

func (w *ProviderEventStream[T]) Current() StreamEvent {
    return w.current
}

func (w *ProviderEventStream[T]) Err() error {
    return w.err
}

func (w *ProviderEventStream[T]) Close() error {
    return w.original.Close()
}
