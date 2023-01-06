package fauna

import (
	"encoding/json"
	"net/http"
	"time"
)

// Stats represents the metrics returned in a Response
type Stats struct {
	ReadOps    int           `json:"read_ops"`
	WriteOps   int           `json:"write_ops"`
	ComputeOps int           `json:"compute_ops"`
	QueryTime  time.Duration `json:"query_time"`
}

// Response represent a standard response from Fauna request
type Response struct {
	Raw        *http.Response
	Bytes      []byte
	Data       json.RawMessage `json:"data"`
	Error      *ServiceError   `json:"error,omitempty"`
	Summary    string          `json:"summary"`
	StaticType string          `json:"static_type"`
	TxnTime    time.Time       `json:"txn_time"`
	Stats      *Stats          `json:"stats,omitempty"`
}
