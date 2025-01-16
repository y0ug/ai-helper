package ssestream

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/y0ug/ai-helper/pkg/llmclient/common"
)

type Decoder interface {
	Event() Event
	Next() bool
	Close() error
	Err() error
}

func NewDecoder(res *http.Response) Decoder {
	if res == nil || res.Body == nil {
		return nil
	}

	var decoder Decoder
	contentType := res.Header.Get("content-type")
	if t, ok := decoderTypes[contentType]; ok {
		decoder = t(res.Body)
	} else {
		scanner := bufio.NewScanner(res.Body)
		decoder = &eventStreamDecoder{rc: res.Body, scn: scanner}
	}
	return decoder
}

var decoderTypes = map[string](func(io.ReadCloser) Decoder){}

func RegisterDecoder(contentType string, decoder func(io.ReadCloser) Decoder) {
	decoderTypes[strings.ToLower(contentType)] = decoder
}

type Event struct {
	Type string
	Data []byte
}

// A base implementation of a Decoder for text/event-stream.
type eventStreamDecoder struct {
	evt Event
	rc  io.ReadCloser
	scn *bufio.Scanner
	err error
}

func (s *eventStreamDecoder) Next() bool {
	if s.err != nil {
		return false
	}

	event := ""
	data := bytes.NewBuffer(nil)

	for s.scn.Scan() {
		txt := s.scn.Bytes()

		// Dispatch event on an empty line
		if len(txt) == 0 {
			s.evt = Event{
				Type: event,
				Data: data.Bytes(),
			}
			return true
		}

		// Split a string like "event: bar" into name="event" and value=" bar".
		name, value, _ := bytes.Cut(txt, []byte(":"))

		// Consume an optional space after the colon if it exists.
		if len(value) > 0 && value[0] == ' ' {
			value = value[1:]
		}

		switch string(name) {
		case "":
			// An empty line in the for ": something" is a comment and should be ignored.
			continue
		case "event":
			event = string(value)
		case "data":
			_, s.err = data.Write(value)
			if s.err != nil {
				break
			}
			_, s.err = data.WriteRune('\n')

		}
	}

	return false
}

func (s *eventStreamDecoder) Event() Event {
	return s.evt
}

func (s *eventStreamDecoder) Close() error {
	return s.rc.Close()
}

func (s *eventStreamDecoder) Err() error {
	return s.err
}

// AnthropicStreamHandler implements BaseStreamHandler for Anthropic's streaming responses
type AnthropicStreamHandler[T any] struct{}

func NewAnthropicStreamHandler[T any]() *AnthropicStreamHandler[T] {
    return &AnthropicStreamHandler[T]{}
}

func (h *AnthropicStreamHandler[T]) HandleEvent(event Event) (T, error) {
    var result T
    
    switch event.Type {
    case "completion":
        if err := json.Unmarshal(event.Data, &result); err != nil {
            return result, err
        }
    case "message_start",
        "message_delta",
        "message_stop",
        "content_block_start",
        "content_block_delta",
        "content_block_stop":
        if err := json.Unmarshal(event.Data, &result); err != nil {
            return result, err
        }
    case "error":
        return result, fmt.Errorf("received error while streaming: %s", string(event.Data))
    }
    
    return result, nil
}

func (h *AnthropicStreamHandler[T]) ShouldContinue(event Event) bool {
    if event.Type == "ping" {
        return true
    }
    return event.Type != "error"
}

func NewAnthropicStream[T any](decoder Decoder, err error) common.Streamer[T] {
    if err != nil {
        return NewBaseStream[T](decoder, &AnthropicStreamHandler[T]{})
    }
    return NewBaseStream[T](decoder, &AnthropicStreamHandler[T]{})
}
