package highlighter

import (
	"bufio"
	"bytes"
	"context"
	"log"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Highlighter handles syntax highlighting for markdown and code blocks
type Highlighter struct {
	writer       *bufio.Writer
	inCodeBlock  bool
	lexer        chroma.Lexer
	defaultLexer chroma.Lexer
	formatter    chroma.Formatter
	style        *chroma.Style
}

// NewHighlighter creates a new instance of Highlighter
func NewHighlighter(writer *bufio.Writer) *Highlighter {
	lexer := lexers.Get("markdown")
	if lexer == nil {
		lexer = lexers.Fallback
	}

	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	return &Highlighter{
		writer:       writer,
		lexer:        lexer,
		defaultLexer: lexer,
		formatter:    formatter,
		style:        style,
	}
}

// ProcessLine handles the highlighting of a single line of text
func (h *Highlighter) ProcessLine(line string) {
	if strings.HasPrefix(line, "```") {
		h.handleCodeBlockMarker(line)
		return
	}
	h.highlightAndPrint(line)
	h.writer.Flush()
}

// handleCodeBlockMarker processes the start/end of code blocks
func (h *Highlighter) handleCodeBlockMarker(line string) {
	h.inCodeBlock = !h.inCodeBlock
	if h.inCodeBlock {
		line = strings.ToLower(line)
		h.highlightAndPrint(line)
		language := strings.Trim(strings.ToLower(line[3:]), "\n")
		h.lexer = lexers.Get(language)
		if h.lexer == nil {
			h.lexer = h.defaultLexer
		}
	} else {
		h.lexer = h.defaultLexer
		h.highlightAndPrint(line)
	}
}

// highlightAndPrint performs the actual syntax highlighting
func (h *Highlighter) highlightAndPrint(line string) {
	iterator, err := h.lexer.Tokenise(nil, line)
	if err != nil {
		log.Printf("Tokenization error: %v", err)
		return
	}

	err = h.formatter.Format(h.writer, h.style, iterator)
	if err != nil {
		log.Printf("Formatting error: %v", err)
		return
	}
}

// ProcessStream processes a stream of text from a channel and highlights it
func (h *Highlighter) ProcessStream(ctx context.Context, ch <-chan string) error {
	defer h.writer.Flush()

	var buffer bytes.Buffer
	var codeFenceBuffer bytes.Buffer
	inPartialCodeFence := false

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case content, ok := <-ch:
			if !ok {
				// Channel closed, process remaining content
				remaining := buffer.String()
				if len(remaining) > 0 {
					h.highlightAndPrint(remaining)
					h.writer.Flush()
				}
				return nil
			}

			// Check for partial code fence
			if inPartialCodeFence {
				codeFenceBuffer.WriteString(content)
				if codeFenceBuffer.Len() >= 3 {
					str := codeFenceBuffer.String()
					if strings.HasPrefix(str, "```") {
						// We found a complete code fence
						h.handleCodeBlockMarker(str)
						buffer.Reset()
						codeFenceBuffer.Reset()
						inPartialCodeFence = false
						continue
					} else if codeFenceBuffer.Len() > 3 {
						// Not a code fence, print buffered content
						buffer.Write(codeFenceBuffer.Bytes())
						codeFenceBuffer.Reset()
						inPartialCodeFence = false
					}
				}
			} else if strings.Contains(content, "`") {
				codeFenceBuffer.WriteString(content)
				inPartialCodeFence = true
				continue
			}

			// Normal content processing
			buffer.WriteString(content)

			// Process complete lines if we have them
			currentBuffer := buffer.String()
			if strings.Contains(currentBuffer, "\n") {
				lines := strings.Split(currentBuffer, "\n")
				for i := 0; i < len(lines)-1; i++ {
					h.ProcessLine(lines[i] + "\n")
				}
				buffer.Reset()
				buffer.WriteString(lines[len(lines)-1])
			} else {
				// Immediately print partial line in real-time
				h.highlightAndPrint(content)
				h.writer.Flush()
			}
		}
	}
}

// ProcessStream processes a stream of text from a channel
// and push it as stream of line
// ProcessStreamToNewLine processes a stream of text and splits it into lines
// sending each complete line to the output channel
func ProcessStreamToNewLine(ctx context.Context, in <-chan string, out chan<- string) error {
	defer close(out)

	var buffer bytes.Buffer
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case content, ok := <-in:
			if !ok {
				// Process remaining content before returning
				remaining := buffer.String()
				if len(remaining) > 0 {
					if !strings.HasSuffix(remaining, "\n") {
						remaining += "\n"
					}
					out <- remaining
				}
				return nil
			}

			buffer.WriteString(content)
			for {
				currentBuffer := buffer.String()
				index := strings.Index(currentBuffer, "\n")
				if index == -1 {
					break
				}
				line := currentBuffer[:index+1]
				buffer.Next(index + 1)
				out <- line
			}
		}
	}

	// remaining := buffer.String()
	// if len(remaining) > 0 {
	// 	if !strings.HasSuffix(remaining, "\n") {
	// 		remaining += "\n"
	// 	}
	// 	out <- remaining
	//
	// }
}
