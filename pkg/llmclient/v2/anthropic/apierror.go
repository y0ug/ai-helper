package anthropic

import (
	"encoding/json"
	"net/http"

	"github.com/y0ug/ai-helper/pkg/llmclient/v2/apierror"
)

func NewAPIErrorAnthropic(resp *http.Response, req *http.Request) apierror.APIError {
	return &APIErrorAnthropic{
		APIErrorBase: apierror.APIErrorBase{
			StatusCode: resp.StatusCode,
			Request:    req,
			Response:   resp,
		},
	}
}

type APIErrorAnthropic struct {
	apierror.APIErrorBase
	ExtraFields map[string]interface{} `json:"-"`
}

func (r *APIErrorAnthropic) UnmarshalJSON(data []byte) (err error) {
	r.JSON = string(data)
	r.ExtraFields = make(map[string]interface{})
	return json.Unmarshal(data, &r.ExtraFields)
}
