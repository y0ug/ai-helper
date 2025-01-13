package openai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
)

// Error represents an error that originates from the API, i.e. when a request is
// made and the API returns a response with a HTTP status code. Other errors are
// not wrapped by this SDK.
type Error struct {
	Code       string `json:"code,required,nullable"`
	Message    string `json:"message,required"`
	Param      string `json:"param,required,nullable"`
	Type       string `json:"type,required"`
	JSON       string `json:"-"`
	StatusCode int
	Request    *http.Request
	Response   *http.Response
}

func (r *Error) UnmarshalJSON(data []byte) (err error) {
	r.JSON = string(data)
	type Alias Error
	return json.Unmarshal(data, (*Alias)(r))
}

func (r *Error) Error() string {
	// Attempt to re-populate the response body
	return fmt.Sprintf(
		"%s \"%s\": %d %s %s",
		r.Request.Method,
		r.Request.URL,
		r.Response.StatusCode,
		http.StatusText(r.Response.StatusCode),
		r.JSON,
	)
}

func (r *Error) DumpRequest(body bool) []byte {
	if r.Request.GetBody != nil {
		r.Request.Body, _ = r.Request.GetBody()
	}
	out, _ := httputil.DumpRequestOut(r.Request, body)
	return out
}

func (r *Error) DumpResponse(body bool) []byte {
	out, _ := httputil.DumpResponse(r.Response, body)
	return out
}
