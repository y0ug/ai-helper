package ai

type Provider interface {
	SetMaxTokens(maxTokens int)
	SetTools(tool []AITools)
	GenerateResponse(messages []Message, toolOutputs ...ToolOutput) (Response, error)
}
