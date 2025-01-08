package ai

type Provider interface {
	GenerateResponse(messages []Message) (Response, error)
}
