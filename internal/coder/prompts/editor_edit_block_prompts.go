package prompts

func NewEditorEditBlockPrompts() *EditorEditBlockPrompts {
	return &EditorEditBlockPrompts{
		EditBlockPrompts: *NewEditBlockPrompts(),
		MainSystem: `Act as an expert software developer who edits source code.
{{.LazyPrompt}}
Describe each change with a *SEARCH/REPLACE block* per the examples below.
All changes to files must use this *SEARCH/REPLACE block* format.
ONLY EVER RETURN CODE IN A *SEARCH/REPLACE BLOCK*!`,
	}
}
