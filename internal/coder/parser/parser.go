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
		fileBlockRegex: regexp.MustCompile(`(?ms)^([^\n]+)\n(?:<source>(\w*)\n(.*?)\n</source>|` + "```" + `(\w*)\n(.*?)\n` + "```" + `)`),
	}
}

func (p *Parser) ParseResponse(response string) []diff.Section {
	var sections []diff.Section

	matches := p.fileBlockRegex.FindAllStringSubmatch(response, -1)

	for _, match := range matches {
		filename := strings.TrimSpace(match[1])
		var language, content string
		
		if match[2] != "" { // <source> format
			language = match[2]
			content = match[3]
		} else { // ```language format
			language = match[4]
			content = match[5]
		}

		section := p.parseSection(filename, language, content)
		if section != nil {
			sections = append(sections, *section)
		}
	}

	return sections
}

func (p *Parser) parseSection(filename, language, content string) *diff.Section {
	searchStart := strings.Index(content, diff.SearchMarker)
	if searchStart == -1 {
		return nil
	}

	separatorStart := strings.Index(content, diff.SeparatorMarker)
	if separatorStart == -1 {
		return nil
	}

	replaceStart := strings.Index(content, diff.ReplaceMarker)
	if replaceStart == -1 {
		return nil
	}

	// Extract the search block (between SEARCH marker and separator)
	searchBlock := strings.TrimSpace(content[searchStart+len(diff.SearchMarker):separatorStart])

	// Extract the replace block (between separator and REPLACE marker)
	replaceBlock := strings.TrimSpace(content[separatorStart+len(diff.SeparatorMarker):replaceStart])

	return &diff.Section{
		Filename:     filename,
		SearchBlock:  searchBlock,
		ReplaceBlock: replaceBlock,
		Language:     language,
	}
}
