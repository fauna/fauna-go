package fauna

import (
	"encoding/json"
	"net/http"
	"time"
)

type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Response struct {
	Raw        *http.Response
	Data       json.RawMessage `json:"data"`
	Error      *ResponseError  `json:"error,omitempty"`
	Summary    string          `json:"summary"`
	StaticType string          `json:"static_type"`
	TxnTime    time.Time       `json:"txn_time"`
}
