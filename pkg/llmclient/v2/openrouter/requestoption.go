package openrouter

import (
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestoption"
)

func WithEnvironmentProduction() requestoption.RequestOption {
	return requestoption.WithBaseURL("https://openrouter.ai/api/v1/")
}
