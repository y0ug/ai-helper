package highlighter

import (
	"bufio"
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestProcessStream(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name: "simple text",
			input: []string{
				"Hello",
				" World\n",
			},
			expected: "Hello World\n",
		},
		{
			name: "code block",
			input: []string{
				"```python\n",
				"print('hello')\n",
				"```\n",
			},
			expected: "```python\nprint('hello')\n```\n",
		},
		{
			name: "split code block marker",
			input: []string{
				"``",
				"`python\n",
				"print('hello')\n",
				"```\n",
			},
			expected: "```python\nprint('hello')\n```\n",
		},
		{
			name: "multiple lines with code block",
			input: []string{
				"Some text\n",
				"```python\n",
				"print('hello')\n",
				"```\n",
				"More text\n",
			},
			expected: "Some text\n```python\nprint('hello')\n```\nMore text\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := bufio.NewWriter(&buf)
			h := NewHighlighter(writer)

			ch := make(chan string)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// Start processing in a goroutine
			go func() {
				for _, chunk := range tt.input {
					ch <- chunk
				}
				close(ch)
			}()

			err := h.ProcessStream(ctx, ch)
			if err != nil {
				t.Errorf("ProcessStream returned error: %v", err)
			}

			writer.Flush()
			result := buf.String()

			// Compare the content ignoring ANSI color codes
			cleanResult := stripANSI(result)
			if cleanResult != tt.expected {
				t.Errorf("ProcessStream() got = %q, want %q", cleanResult, tt.expected)
			}
		})
	}
}

// stripANSI removes ANSI escape sequences from the string
func stripANSI(str string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range str {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
