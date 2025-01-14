package anthropic

import (
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestconfig"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestoption"
)

func WithEnvironmentProduction() requestoption.RequestOption {
	return requestoption.WithBaseURL("https://api.anthropic.com/")
}

func WithApiVersionAnthropic() requestoption.RequestOption {
	return func(r *requestconfig.RequestConfig) error {
		return r.Apply(requestoption.WithHeader("anthropic-version", "2023-06-01"))
	}
}
