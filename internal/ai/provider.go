package ai

type Provider interface {
	Settings() AIModelSettings
	// SetMaxTokens(maxTokens int)
	// SetTools(tool []AITools)
	// SetStream(stream bool)
	GenerateResponse(messages []AIMessage) (AIResponse, error)
}
