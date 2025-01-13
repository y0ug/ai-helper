package ssestream

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

// sseFramer implements the jsonrpc2.Framer interface for SSE
type sseFramer struct{}

// sseReader reads JSON-RPC messages from an SSE stream
type sseReader struct {
	scanner *bufio.Scanner
}

// sseWriter writes JSON-RPC messages as SSE events
type sseWriter struct {
	writer io.Writer
}

// Reader returns a new sseReader for the given io.Reader
func (sseFramer) Reader(r io.Reader) jsonrpc2.Reader {
	scanner := bufio.NewScanner(r)
	scanner.Split(splitSSE)
	return &sseReader{scanner: scanner}
}

// Writer returns a new sseWriter for the given io.Writer
func (sseFramer) Writer(w io.Writer) jsonrpc2.Writer {
	return &sseWriter{writer: w}
}

// Read parses the SSE stream and extracts JSON-RPC messages
func (sr *sseReader) Read(ctx context.Context) (jsonrpc2.Message, int64, error) {
	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		default:
		}

		// Read the next SSE event
		if !sr.scanner.Scan() {
			if err := sr.scanner.Err(); err != nil {
				return nil, 0, fmt.Errorf("SSE read error: %w", err)
			}
			return nil, 0, io.EOF
		}

		event := sr.scanner.Text()
		// Extract the data field
		data, err := parseSSE(event)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to parse SSE event: %w", err)
		}
		if data == "" {
			continue // Ignore events without data
		}

		// Unmarshal the JSON-RPC message
		var raw jsonrpc2.RawMessage
		if err := raw.UnmarshalJSON([]byte(data)); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal JSON-RPC message: %w", err)
		}

		msg, err := jsonrpc2.DecodeMessage(raw)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to decode JSON-RPC message: %w", err)
		}

		// Calculate the length of the data
		length := int64(len(data))
		return msg, length, nil
	}
}

// Write sends a JSON-RPC message as an SSE event
func (sw *sseWriter) Write(ctx context.Context, msg jsonrpc2.Message) (int64, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	// Marshal the JSON-RPC message
	data, err := jsonrpc2.EncodeMessage(msg)
	if err != nil {
		return 0, fmt.Errorf("marshaling JSON-RPC message: %w", err)
	}

	// Prepare the SSE event
	var sb strings.Builder
	sb.WriteString("data: ")
	sb.Write(data)
	sb.WriteString("\n\n") // SSE messages are terminated by a double newline

	eventData := sb.String()

	// Write the SSE event
	n, err := sw.writer.Write([]byte(eventData))
	if err != nil {
		return 0, fmt.Errorf("failed to write SSE event: %w", err)
	}

	return int64(n), nil
}

// splitSSE is a bufio.SplitFunc that splits the input into SSE events
func splitSSE(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// SSE events are separated by a double newline
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := strings.Index(string(data), "\n\n"); i >= 0 {
		// We have a full event
		return i + 2, data[0:i], nil
	}
	if atEOF {
		// Return the remaining data
		return len(data), data, nil
	}
	// Request more data
	return 0, nil, nil
}

// parseSSE extracts the data field from an SSE event
func parseSSE(event string) (string, error) {
	var dataLines []string
	for _, line := range strings.Split(event, "\n") {
		// Ignore comments and empty lines
		if strings.HasPrefix(line, ":") || strings.TrimSpace(line) == "" {
			continue
		}
		// Parse lines starting with "data:"
		if strings.HasPrefix(line, "data:") {
			// Remove the "data:" prefix and any leading whitespace
			data := strings.TrimSpace(line[5:])
			dataLines = append(dataLines, data)
		}
	}
	// Join multiple data lines with newline as per SSE spec
	return strings.Join(dataLines, "\n"), nil
}
