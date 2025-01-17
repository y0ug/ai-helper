package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// LoggingMiddleware creates a middleware that logs request and response details
func TimeitMiddleware(
	logger zerolog.Logger,
) func(*http.Request, func(*http.Request) (*http.Response, error)) (*http.Response, error) {
	return func(req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error) {
		start := time.Now()

		// Call the next middleware/handler
		resp, err := next(req)
		if err != nil {
			return resp, err
		}

		end := time.Now()

		logger.Debug().Dur("time", end.Sub(start)).Msg("Request took")

		return resp, err
	}
}
