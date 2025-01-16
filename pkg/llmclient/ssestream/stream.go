package ssestream

// StreamHandler defines the interface for provider-specific stream handling
type StreamHandler[T any] interface {
	HandleEvent(event Event) (T, error)
	ShouldContinue(event Event) bool
}

// Stream provides core streaming functionality that can be reused across providers
type Stream[T any] struct {
	decoder Decoder
	handler StreamHandler[T]
	current T
	err     error
	done    bool
}

func NewStream[T any](decoder Decoder, handler StreamHandler[T]) *Stream[T] {
	return &Stream[T]{
		decoder: decoder,
		handler: handler,
		done:    false,
	}
}

func (s *Stream[T]) Next() bool {
	if s.err != nil || s.done {
		return false
	}

	for s.decoder.Next() {
		if !s.handler.ShouldContinue(s.decoder.Event()) {
			s.done = true
			return false
		}

		current, err := s.handler.HandleEvent(s.decoder.Event())
		if err != nil {
			s.err = err
			return false
		}

		s.current = current
		return true
	}

	if err := s.decoder.Err(); err != nil {
		s.err = err
	}
	return false
}

func (s *Stream[T]) Current() T {
	return s.current
}

func (s *Stream[T]) Err() error {
	return s.err
}

func (s *Stream[T]) Close() error {
	return s.decoder.Close()
}
