package common

import "github.com/y0ug/ai-helper/pkg/llmclient/stream"

// EventStream represents a normalized stream event across providers
type EventStream struct {
	Type    string // text_delta, message_start, message_stop, etc
	Delta   interface{}
	Message *ChatMessage
}

// NewProviderEventStream creates a new stream that normalizes provider events
func NewProviderEventStream[TypeIn any](
	decoder stream.Streamer[TypeIn],
	handler stream.StreamHandler[EventStream, TypeIn],
) stream.Streamer[EventStream] {
	return stream.NewStream(decoder, handler)
}
