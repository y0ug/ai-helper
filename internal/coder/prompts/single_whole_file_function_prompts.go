package prompts

func NewSingleWholeFileFunctionPrompts() *SingleWholeFileFunctionPrompts {
	return &SingleWholeFileFunctionPrompts{
		BasePrompts: *NewBasePrompts(),
		MainSystem: `Act as an expert software developer.
Take requests for changes to the supplied code.
If the request is ambiguous, ask questions.

Once you understand the request you MUST use the 'write_file' function to update the file to make the changes.`,
		SystemReminder: `ONLY return code using the 'write_file' function.
NEVER return code outside the 'write_file' function.`,
		RedactedEditMessage: "No changes are needed.",
	}
}
