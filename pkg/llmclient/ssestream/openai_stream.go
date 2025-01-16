package ssestream

import (
	"encoding/json"
	"fmt"

	"github.com/y0ug/ai-helper/pkg/llmclient/common"
)

// OpenAIStreamHandler implements BaseStreamHandler for OpenAI's streaming responses
type OpenAIStreamHandler[T any] struct{}

func NewOpenAIStreamHandler[T any]() *OpenAIStreamHandler[T] {
	return &OpenAIStreamHandler[T]{}
}

func (h *OpenAIStreamHandler[T]) HandleEvent(event Event) (T, error) {
	var result T

	if len(event.Data) == 0 {
		return result, nil
	}

	if err := json.Unmarshal(event.Data, &result); err != nil {
		return result, fmt.Errorf("error unmarshalling OpenAI event: %w", err)
	}

	return result, nil
}

func (h *OpenAIStreamHandler[T]) ShouldContinue(event Event) bool {
	if string(event.Data) == "[DONE]" {
		return false
	}
	return true
}

func NewOpenAIStream[T any](decoder Decoder, err error) common.Streamer[T] {
	if err != nil {
		return NewBaseStream[T](decoder, &OpenAIStreamHandler[T]{})
	}
	return NewBaseStream[T](decoder, &OpenAIStreamHandler[T]{})
}
