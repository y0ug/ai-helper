package parser

import (
	"regexp"
	"strings"

	"github.com/y0ug/ai-helper/internal/coder/diff"
)

type Parser struct {
	fileBlockRegex *regexp.Regexp
}

func New() *Parser {
	return &Parser{
		fileBlockRegex: regexp.MustCompile(`(?ms)^([^\n]+)\n<source>(\w*)\n(.*?)\n</source>`),
	}
}

func (p *Parser) ParseResponse(response string) []diff.Section {
	var sections []diff.Section

	matches := p.fileBlockRegex.FindAllStringSubmatch(response, -1)

	for _, match := range matches {
		filename := strings.TrimSpace(match[1])
		language := match[2]
		content := match[3]

		section := p.parseSection(filename, language, content)
		if section != nil {
			sections = append(sections, *section)
		}
	}

	return sections
}

func (p *Parser) parseSection(filename, language, content string) *diff.Section {
	lines := strings.Split(content, "\n")
	var searchLines, replaceLines []string
	mode := ""

	hasSearchMarker := false
	hasReplaceMarker := false

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, diff.SearchMarker):
			mode = "search"
			hasSearchMarker = true
			continue
		case strings.HasPrefix(line, diff.SeparatorMarker):
			mode = ""
			continue
		case strings.HasPrefix(line, diff.ReplaceMarker):
			mode = "replace"
			hasReplaceMarker = true
			continue
		}

		switch mode {
		case "search":
			searchLines = append(searchLines, line)
		case "replace":
			replaceLines = append(replaceLines, line)
		}
	}

	// Only create a section if we have both markers
	if !hasSearchMarker || !hasReplaceMarker {
		return nil
	}

	return &diff.Section{
		Filename:     filename,
		SearchBlock:  strings.Join(searchLines, "\n"),
		ReplaceBlock: strings.Join(replaceLines, "\n"),
		Language:     language,
	}
}
