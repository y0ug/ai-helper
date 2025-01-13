package llmclient

//go:generate go run go.uber.org/mock/mockgen@latest -destination=mock.go -package=ai github.com/y0ug/ai-helper/internal/ai AIClient,Provider,AIConversation,InfoProvider
