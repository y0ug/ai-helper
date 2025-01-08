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
		fileBlockRegex: regexp.MustCompile(`(?m)^([^\n]+)\n\x60\x60\x60(\w*)\n(.*?)\n\x60\x60\x60`),
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
	inSearch := false
	inReplace := false

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, diff.SearchMarker):
			inSearch = true
			inReplace = false
		case strings.HasPrefix(line, diff.SeparatorMarker):
			inSearch = false
			inReplace = false
		case strings.HasPrefix(line, diff.ReplaceMarker):
			inSearch = false
			inReplace = true
		default:
			if inSearch {
				searchLines = append(searchLines, line)
			} else if inReplace {
				replaceLines = append(replaceLines, line)
			}
		}
	}

	if len(searchLines) == 0 && len(replaceLines) == 0 {
		return nil
	}

	return &diff.Section{
		Filename:     filename,
		SearchBlock:  strings.Join(searchLines, "\n"),
		ReplaceBlock: strings.Join(replaceLines, "\n"),
		Language:     language,
	}
}
