package llmclient

//go:generate go run go.uber.org/mock/mockgen@latest -destination=mock.go -package=llmclient github.com/y0ug/ai-helper/pkg/llmclient Provider,AIClient,InfoProvider
