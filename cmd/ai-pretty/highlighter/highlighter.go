package highlighter

import (
	"bufio"
	"bytes"
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

// ProcessStream processes a stream of text from a channel
func ProcessStream(ch <-chan string, writer *bufio.Writer) {
	defer writer.Flush()

	h := NewHighlighter(writer)
	var buffer bytes.Buffer

	for content := range ch {
		buffer.WriteString(content)
		for {
			currentBuffer := buffer.String()
			index := strings.Index(currentBuffer, "\n")
			if index == -1 {
				break
			}
			line := currentBuffer[:index+1]
			buffer.Next(index + 1)
			h.ProcessLine(line)
		}
	}

	remaining := buffer.String()
	if len(remaining) > 0 {
		if !strings.HasSuffix(remaining, "\n") {
			remaining += "\n"
		}
		h.ProcessLine(remaining)
	}
}
