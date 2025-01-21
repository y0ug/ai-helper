package prompts

func NewEditorWholeFilePrompts() *EditorWholeFilePrompts {
	return &EditorWholeFilePrompts{
		WholeFilePrompts: *NewWholeFilePrompts(),
		MainSystem: `Act as an expert software developer and make changes to source code.
{{.LazyPrompt}}
Output a copy of each file that needs changes.`,
	}
}
