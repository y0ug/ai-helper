package ssestream

// BaseStreamHandler defines the interface for provider-specific stream handling
type BaseStreamHandler[T any] interface {
	HandleEvent(event Event) (T, error)
	ShouldContinue(event Event) bool
}

// BaseStream provides core streaming functionality that can be reused across providers
type BaseStream[T any] struct {
	decoder Decoder
	handler BaseStreamHandler[T]
	current T
	err     error
	done    bool
}

func NewBaseStream[T any](decoder Decoder, handler BaseStreamHandler[T]) *BaseStream[T] {
	return &BaseStream[T]{
		decoder: decoder,
		handler: handler,
		done:    false,
	}
}

func (s *BaseStream[T]) Next() bool {
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

func (s *BaseStream[T]) Current() T {
	return s.current
}

func (s *BaseStream[T]) Err() error {
	return s.err
}

func (s *BaseStream[T]) Close() error {
	return s.decoder.Close()
}
