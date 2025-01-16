package common

// StreamEvent represents a normalized stream event across providers
type StreamEvent struct {
	Type    string // text_delta, message_start, message_stop, etc
	Delta   interface{}
	Message *BaseChatMessage
}

// ProviderEventHandler handles provider-specific event processing
type ProviderEventHandler interface {
	ProcessEvent(data []byte) StreamEvent
}
