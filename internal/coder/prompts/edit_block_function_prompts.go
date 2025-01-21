package prompts

func NewEditBlockFunctionPrompts() *EditBlockFunctionPrompts {
	return &EditBlockFunctionPrompts{
		BasePrompts: *NewBasePrompts(),
		MainSystem: `Act as an expert software developer.
Take requests for changes to the supplied code.
If the request is ambiguous, ask questions.

Once you understand the request you MUST use the 'replace_lines' function to edit the files to make the needed changes.`,
		RedactedEditMessage: "No changes are needed.",
	}
}
