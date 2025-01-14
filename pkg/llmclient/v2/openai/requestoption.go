package openai

import (
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestconfig"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestoption"
)

// WithOrganization returns a RequestOption that sets the client setting "organization".
func WithOrganization(value string) requestoption.RequestOption {
	return func(r *requestconfig.RequestConfig) error {
		r.Organization = value
		return r.Apply(requestoption.WithHeader("OpenAI-Organization", value))
	}
}

// WithProject returns a RequestOption that sets the client setting "project".
func WithProject(value string) requestoption.RequestOption {
	return func(r *requestconfig.RequestConfig) error {
		r.Project = value
		return r.Apply(requestoption.WithHeader("OpenAI-Project", value))
	}
}

// WithEnvironmentProductionOpenAI returns a RequestOption that sets the current
// environment to be the "production" environment. An environment specifies which base URL
// to use by default.
func WithEnvironmentProduction() requestoption.RequestOption {
	return requestoption.WithBaseURL("https://api.openai.com/v1/")
}
