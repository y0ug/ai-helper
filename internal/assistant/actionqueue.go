package assistant

import (
	"github.com/y0ug/ai-helper/internal/assistant/actions"
)

type Queuer[T any] interface {
	Enqueue(actions ...T)
	Dequeue() (T, bool)
	IsEmpty() bool
}

type ActionQueuer = Queuer[actions.Action[any]]

// TODO: Implement mutex
type ActionQueue struct {
	items []actions.Action[any]
}

func NewActionQueue() *ActionQueue {
	return &ActionQueue{
		items: make([]actions.Action[any], 0),
	}
}

func (q *ActionQueue) Enqueue(actions ...actions.Action[any]) {
	q.items = append(q.items, actions...)
}

func (q *ActionQueue) Dequeue() (actions.Action[any], bool) {
	if len(q.items) == 0 {
		return actions.Action[any]{}, false
	}
	action := q.items[0]
	q.items = q.items[1:]
	return action, true
}

func (q *ActionQueue) IsEmpty() bool {
	return len(q.items) == 0
}
