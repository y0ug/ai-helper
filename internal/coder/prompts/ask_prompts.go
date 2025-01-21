package prompts

func NewAskPrompts() *AskPrompts {
	return &AskPrompts{
		BasePrompts: *NewBasePrompts(),
		MainSystem: `Act as an expert code analyst.
Answer questions about the supplied code.
Always reply to the user in {{.Language}}.

Describe code changes however you like. Don't use SEARCH/REPLACE blocks!`,
	}
}
