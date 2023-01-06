package fauna

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ResponseError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func (r ResponseError) Error() string {
	return fmt.Sprintf("%s: %s", r.Code, r.Message)
}

type Stats struct {
	ReadOps    int           `json:"read_ops"`
	WriteOps   int           `json:"write_ops"`
	ComputeOps int           `json:"compute_ops"`
	QueryTime  time.Duration `json:"query_time"`
}

type Response struct {
	Raw        *http.Response
	Bytes      []byte
	Data       json.RawMessage `json:"data"`
	Error      *ResponseError  `json:"error,omitempty"`
	Summary    string          `json:"summary"`
	StaticType string          `json:"static_type"`
	TxnTime    time.Time       `json:"txn_time"`
	Stats      *Stats          `json:"stats,omitempty"`
}
