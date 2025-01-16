package ssestream

// BaseHandler provides a default implementation of BaseStreamHandler
type BaseHandler[T any] struct{}

func NewBaseHandler[T any]() *BaseHandler[T] {
    return &BaseHandler[T]{}
}

func (h *BaseHandler[T]) HandleEvent(event Event) (T, error) {
    var result T
    return result, nil
}

func (h *BaseHandler[T]) ShouldContinue(event Event) bool {
    return true
}
