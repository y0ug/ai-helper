package middleware

import (
	"fmt"
	"net/http"

	"github.com/y0ug/ai-helper/pkg/llmclient/openai/requestconfig"
	"github.com/y0ug/ai-helper/pkg/llmclient/openai/requestoption"
)

func WithRateLimitMiddleware(rateLimit *requestconfig.RateLimit) requestoption.RequestOption {
	return requestoption.WithMiddleware(
		func(req *http.Request, next requestoption.MiddlewareNext) (*http.Response, error) {
			res, err := next(req)
			if err != nil {
				return res, err
			}

			if res == nil {
				return res, fmt.Errorf("received nil response")
			}

			err = rateLimit.Update(res.Header)
			if err != nil {
				// Log the error but don't fail the request
				// You can choose to handle it differently based on your needs
				// fmt.Printf("Failed to update rate limit: %v\n", err)
			}

			return res, nil
		},
	)
}
