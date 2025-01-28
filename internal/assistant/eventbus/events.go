package eventbus

import (
	"time"
)

type EventType int

const (
	EventInput EventType = iota
	EventError
	EventAction
	EventActionResult
	EventActionOutput
	EventActionProcess
	EventOutput
	EventStatusUpdate
	EventShutdown
	EventLLMRequest
)

type Event struct {
	Type     EventType
	Payload  interface{}
	Metadata map[string]string
}

// Generic event constructor
func NewEvent(t EventType, payload interface{}) Event {
	return Event{
		Type:    t,
		Payload: payload,
		Metadata: map[string]string{
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
			"source":    "source", // runtime.Caller(2).Function,
		},
	}
}

type StatusUpdate struct {
	Old string
	New string
}

type UserInput struct {
	Source  string
	Content string
}
