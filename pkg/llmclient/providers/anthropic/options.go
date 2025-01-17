package anthropic

import (
	"github.com/y0ug/ai-helper/pkg/llmclient/http/config"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
)

func WithEnvironmentProduction() options.RequestOption {
	return options.WithBaseURL("https://api.anthropic.com/")
}

func WithApiVersionAnthropic() options.RequestOption {
	return func(r *config.RequestConfig) error {
		return r.Apply(options.WithHeader("anthropic-version", "2023-06-01"))
	}
}
