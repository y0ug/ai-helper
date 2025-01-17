package openai

import (
	"encoding/json"
	"net/http"

	"github.com/y0ug/ai-helper/pkg/llmclient/request/apierror"
)

type APIErrorOpenAI struct {
	apierror.APIErrorBase
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

func NewAPIErrorOpenAI(resp *http.Response, req *http.Request) apierror.APIError {
	return &APIErrorOpenAI{
		APIErrorBase: apierror.APIErrorBase{
			StatusCode: resp.StatusCode,
			Request:    req,
			Response:   resp,
		},
	}
}
