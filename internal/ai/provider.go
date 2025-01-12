package ai

type Provider interface {
	SetMaxTokens(maxTokens int)
	SetTools(tool []AITools)
	SetStream(stream bool)
	GenerateResponse(messages []AIMessage) (AIResponse, error)
}
