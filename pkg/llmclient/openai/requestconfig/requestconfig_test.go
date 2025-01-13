package requestconfig

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestNewRequestConfig(t *testing.T) {
	ctx := context.Background()
	testCases := []struct {
		name        string
		method      string
		url         string
		body        interface{}
		dst         interface{}
		expectError bool
	}{
		{
			name:        "Valid GET request",
			method:      "GET",
			url:         "https://api.example.com/test",
			body:        nil,
			dst:         &struct{}{},
			expectError: false,
		},
		{
			name:        "Valid POST request with body",
			method:      "POST",
			url:         "https://api.example.com/test",
			body:        map[string]string{"key": "value"},
			dst:         &struct{}{},
			expectError: false,
		},
		{
			name:        "Invalid URL",
			method:      "GET",
			url:         "://invalid-url",
			body:        nil,
			dst:         &struct{}{},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := NewRequestConfig(ctx, tc.method, tc.url, tc.body, tc.dst)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if cfg == nil {
				t.Error("Expected config but got nil")
				return
			}

			// Verify request properties
			if cfg.Request.Method != tc.method {
				t.Errorf("Expected method %s, got %s", tc.method, cfg.Request.Method)
			}

			// Check headers
			if cfg.Request.Header.Get("Accept") != "application/json" {
				t.Error("Accept header not set correctly")
			}

			userAgent := cfg.Request.Header.Get("User-Agent")
			if userAgent == "" {
				t.Error("User-Agent header not set")
			}
		})
	}
}

func TestRequestConfigExecute(t *testing.T) {
	ctx := context.Background()
	baseURL, _ := url.Parse("https://api.example.com")

	_ = &RequestConfig{
		Context: ctx,
		Request: &http.Request{
			Method: "GET",
			URL:    &url.URL{Path: "/test"},
		},
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
	}

	// err := cfg.Execute()
	// if err != nil {
	// 	t.Error("Not Expected error")
	// }

	// Test invalid base URL
	invalidCfg := &RequestConfig{
		Context: ctx,
		Request: &http.Request{
			Method: "GET",
			URL:    &url.URL{Path: "/test"},
		},
		BaseURL:    nil,
		HTTPClient: &http.Client{},
	}

	err := invalidCfg.Execute()
	if err == nil {
		t.Error("Expected error for nil base URL, got none")
	}
}

func TestRetryDelay(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}

	// Test with Retry-After header in seconds
	resp.Header.Set("Retry-After", "2")
	delay := retryDelay(resp, 0)
	if delay != 2*time.Second {
		t.Errorf("Expected 2s delay, got %v", delay)
	}

	// Test with Retry-After-Ms header
	resp.Header.Set("Retry-After-Ms", "1000")
	delay = retryDelay(resp, 0)
	if delay != time.Second {
		t.Errorf("Expected 1s delay, got %v", delay)
	}

	// Test exponential backoff
	resp.Header = http.Header{} // Clear headers
	delay = retryDelay(resp, 1)
	if delay > 8*time.Second { // Max delay
		t.Errorf("Delay exceeded maximum: %v", delay)
	}
}

func TestRequestConfigClone(t *testing.T) {
	ctx := context.Background()
	originalCfg := &RequestConfig{
		MaxRetries: 3,
		APIKey:     "test-key",
		Request:    &http.Request{Method: "GET"},
		Context:    ctx,
	}

	newCtx := context.WithValue(ctx, "key", "value")
	clonedCfg := originalCfg.Clone(newCtx)

	if clonedCfg == nil {
		t.Fatal("Clone returned nil")
	}

	if clonedCfg.MaxRetries != originalCfg.MaxRetries {
		t.Error("MaxRetries not cloned correctly")
	}

	if clonedCfg.APIKey != originalCfg.APIKey {
		t.Error("APIKey not cloned correctly")
	}

	if clonedCfg.Context != newCtx {
		t.Error("Context not updated in clone")
	}
}
