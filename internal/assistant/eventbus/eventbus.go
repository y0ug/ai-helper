package eventbus

import (
	"sync"
)

type EventBus struct {
	subscribers  []chan Event
	middleware   []Middleware
	shutdownChan chan struct{}
	mu           sync.RWMutex
}

var (
	instance *EventBus
	once     sync.Once
)

type (
	Middleware   func(next EventHandler) EventHandler
	EventHandler func(Event) error
)

func GetEventBus() *EventBus {
	once.Do(func() {
		instance = &EventBus{
			shutdownChan: make(chan struct{}),
		}
	})
	return instance
}

func (b *EventBus) Subscribe(bufferSize int) <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, bufferSize)
	b.subscribers = append(b.subscribers, ch)
	return ch
}

func (b *EventBus) Unsubscribe(ch <-chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, sub := range b.subscribers {
		// if reflect.DeepEqual(sub, ch) {
		if (<-chan Event)(sub) == ch {
			b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
			close(sub)
			break
		}
	}
}

func (b *EventBus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, sub := range b.subscribers {
		select {
		case sub <- event:
		case <-b.shutdownChan:
			return
		default:
			// Handle overflow (e.g., log or create overflow channel)
		}
	}
}

func (b *EventBus) Use(middleware ...Middleware) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.middleware = append(b.middleware, middleware...)
}

func (b *EventBus) Process(handler EventHandler) EventHandler {
	for i := len(b.middleware) - 1; i >= 0; i-- {
		handler = b.middleware[i](handler)
	}
	return handler
}

func (b *EventBus) Shutdown() {
	close(b.shutdownChan)
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, sub := range b.subscribers {
		close(sub)
	}
	b.subscribers = nil
}
