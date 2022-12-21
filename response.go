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
	TxnTime    time.Time       `json:"txn_time"`
	StaticType string          `json:"static_type"`
	Summary    string          `json:"summary"`
}

func NewResponse() *Response {
	return &Response{
		TxnTime: time.Time{},
	}
}

func ErrorResponse(err error) *Response {
	return &Response{
		err: err,
	}
}
