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
			if s.err != nil {
				break
			}
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

type Streamer[T any] interface {
	Next() bool
	Current() T
	Err() error
	Close() error
}

type BaseStream[T any] struct {
	decoder Decoder
	cur     T
	err     error
	done    bool
}

type AnthropicStream[T any] struct {
	BaseStream[T]
}

func NewStream[T any](decoder Decoder, err error) Streamer[T] {
	return &BaseStream[T]{
		decoder: decoder,
		err:     err,
	}
}

func (s *BaseStream[T]) Next() bool {
	if s.err != nil {
		return false
	}

	for s.decoder.Next() {
		if s.done {
			return false
		}

		if bytes.HasPrefix(s.decoder.Event().Data, []byte("[DONE]")) {
			s.done = true
			return false
		}

		if s.decoder.Event().Type == "" {
			ep := gjson.GetBytes(s.decoder.Event().Data, "error")
			if ep.Exists() {
				s.err = fmt.Errorf("received error while streaming: %s", ep.String())
				return false
			}
			s.err = json.Unmarshal(s.decoder.Event().Data, &s.cur)
			if s.err != nil {
				return false
			}
			return true
		} else {
			ep := gjson.GetBytes(s.decoder.Event().Data, "error")
			if ep.Exists() {
				s.err = fmt.Errorf("received error while streaming: %s", ep.String())
				return false
			}
			s.err = json.Unmarshal([]byte(fmt.Sprintf(`{ "event": %q, "data": %s }`, s.decoder.Event().Type, s.decoder.Event().Data)), &s.cur)
			if s.err != nil {
				return false
			}
			return true
		}
	}

	return false
}

func (s *BaseStream[T]) Current() T {
	return s.cur
}

func (s *BaseStream[T]) Err() error {
	return s.err
}

func (s *BaseStream[T]) Close() error {
	return s.decoder.Close()
}

func NewAnthropicStream[T any](decoder Decoder, err error) Streamer[T] {
	return &AnthropicStream[T]{
		BaseStream: BaseStream[T]{
			decoder: decoder,
			err:     err,
		},
	}
}

func (s *AnthropicStream[T]) Next() bool {
	if s.err != nil {
		return false
	}

	for s.decoder.Next() {
		switch s.decoder.Event().Type {
		case "completion":
			s.err = json.Unmarshal(s.decoder.Event().Data, &s.cur)
			if s.err != nil {
				return false
			}
			return true
		case "message_start", "message_delta", "message_stop", "content_block_start", "content_block_delta", "content_block_stop":
			s.err = json.Unmarshal(s.decoder.Event().Data, &s.cur)
			if s.err != nil {
				return false
			}
			return true
		case "ping":
			continue
		case "error":
			s.err = fmt.Errorf("received error while streaming: %s", string(s.decoder.Event().Data))
			return false
		}
	}

	return false
}
