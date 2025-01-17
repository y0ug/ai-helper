package common

import "github.com/y0ug/ai-helper/pkg/llmclient/http/streaming"

// EventStream represents a normalized stream event across providers
type EventStream struct {
	Type    string // text_delta, message_start, message_stop, etc
	Delta   interface{}
	Message *ChatMessage
}

// NewProviderEventStream creates a new stream that normalizes provider events
func NewProviderEventStream[TypeIn any](
	decoder streaming.Streamer[TypeIn],
	handler streaming.StreamHandler[EventStream, TypeIn],
) streaming.Streamer[EventStream] {
	return streaming.NewStream(decoder, handler)
}
