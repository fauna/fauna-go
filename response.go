package fauna

import (
	"encoding/json"
	"net/http"
	"time"
)

type Response struct {
	raw        *http.Response
	err        error
	Data       json.RawMessage `json:"data"`
	Error      ServiceError    `json:"error"`
	Summary    string          `json:"summary"`
	StaticType string          `json:"static_type"`
	TxnTime    time.Time       `json:"txn_time"`
}

func NewResponse() *Response {
	return &Response{
		TxnTime: time.Time{},
	}
}

func (r *Response) String() string {
	j, e := json.Marshal(r)
	if e != nil {
		return ""
	}
	return string(j)
}

func ErrorResponse(err error) *Response {
	return &Response{
		err: err,
	}
}
