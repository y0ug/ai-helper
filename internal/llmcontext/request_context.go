// FileContent represents a loaded file's content
type FileContent struct {
	Path    string
	Content string
}

// FormatFiles returns a formatted string of all loaded files
func (rc *RequestContext) FormatFiles() string {
	var sb strings.Builder
	for path, content := range rc.files {
		sb.WriteString(fmt.Sprintf("\n%s\n```\n%s\n```\n", path, content))
	}
	return sb.String()
}

// GetTemplateFuncs returns template functions
func (rc *RequestContext) GetTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"Files": rc.FormatFiles,
	}
}
