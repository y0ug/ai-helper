package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"time"
)

// LoggingMiddleware creates a middleware that logs request and response details
func LoggingMiddleware() func(*http.Request, func(*http.Request) (*http.Response, error)) (*http.Response, error) {
	return func(req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error) {
		// Log request
		reqDump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			fmt.Printf("Error dumping request: %v\n", err)
		} else {
			fmt.Printf("Request:\n%s\n", string(reqDump))
		}

		start := time.Now()

		// Call the next middleware/handler
		resp, err := next(req)
		if err != nil {
			return resp, err
		}

		end := time.Now()

		fmt.Printf("Request took %v\n", end.Sub(start))

		// Log response
		if resp != nil {
			// Read and restore the response body
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			respDump, err := httputil.DumpResponse(resp, true)
			if err != nil {
				fmt.Printf("Error dumping response: %v\n", err)
			} else {
				fmt.Printf("Response:\n%s\n", string(respDump))
			}

			// Restore the response body again for subsequent readers
			resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		return resp, err
	}
}
