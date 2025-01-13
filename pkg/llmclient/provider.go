package llmclient

type Provider interface {
	Settings() AIModelSettings
	GenerateResponse(messages []AIMessage) (AIResponse, error)
}
