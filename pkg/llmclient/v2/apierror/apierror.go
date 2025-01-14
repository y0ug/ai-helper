package apierror

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
)

type APIError interface {
	Error() string
	UnmarshalJSON(data []byte) error
	DumpRequest(body bool) []byte
	DumpResponse(body bool) []byte
}

type APIErrorBase struct {
	JSON       string `json:"-"`
	StatusCode int
	Request    *http.Request
	Response   *http.Response
}

// Error represents an error that originates from the API, i.e. when a request is
// made and the API returns a response with a HTTP status code. Other errors are
// not wrapped by this SDK.

func (r *APIErrorBase) UnmarshalJSON(data []byte) (err error) {
	r.JSON = string(data)
	type Alias APIErrorBase
	return json.Unmarshal(data, (*Alias)(r))
}

func (r *APIErrorBase) Error() string {
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

func (r *APIErrorBase) DumpRequest(body bool) []byte {
	if r.Request.GetBody != nil {
		r.Request.Body, _ = r.Request.GetBody()
	}
	out, _ := httputil.DumpRequestOut(r.Request, body)
	return out
}

func (r *APIErrorBase) DumpResponse(body bool) []byte {
	out, _ := httputil.DumpResponse(r.Response, body)
	return out
}

type APIErrorOpenAI struct {
	APIErrorBase
	Code    string `json:"code,required,nullable"`
	Message string `json:"message,required"`
	Param   string `json:"param,required,nullable"`
	Type    string `json:"type,required"`
	JSON    string `json:"-"`
}

func (r *APIErrorOpenAI) UnmarshalJSON(data []byte) (err error) {
	r.JSON = string(data)
	type Alias APIErrorOpenAI
	return json.Unmarshal(data, (*Alias)(r))
}

func NewAPIErrorOpenAI(resp *http.Response, req *http.Request) APIError {
	return &APIErrorOpenAI{
		APIErrorBase: APIErrorBase{
			StatusCode: resp.StatusCode,
			Request:    req,
			Response:   resp,
		},
	}
}

func NewAPIErrorAnthropic(resp *http.Response, req *http.Request) APIError {
	return &APIErrorAnthropic{
		APIErrorBase: APIErrorBase{
			StatusCode: resp.StatusCode,
			Request:    req,
			Response:   resp,
		},
	}
}

type APIErrorAnthropic struct {
	APIErrorBase
	ExtraFields map[string]interface{} `json:"-"`
}

func (r *APIErrorAnthropic) UnmarshalJSON(data []byte) (err error) {
	r.JSON = string(data)
	r.ExtraFields = make(map[string]interface{})
	return json.Unmarshal(data, &r.ExtraFields)
}
