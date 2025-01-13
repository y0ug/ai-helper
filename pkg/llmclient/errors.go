package llmclient

import "fmt"

// APIError represents an error returned by the AI provider's API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API Error %d: %s", e.StatusCode, e.Message)
}

// NewAPIError creates a new APIError instance.
func NewAPIError(statusCode int, message string) error {
	return &APIError{
		StatusCode: statusCode,
		Message:    message,
	}
}
