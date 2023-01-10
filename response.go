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
	Bytes      []byte
	Data       json.RawMessage `json:"data"`
	Error      *ServiceError   `json:"error,omitempty"`
	Logging    []string        `json:"logging,omitempty"`
	Raw        *http.Response
	StaticType string    `json:"static_type"`
	Stats      *Stats    `json:"stats,omitempty"`
	Summary    string    `json:"summary"`
	TxnTime    time.Time `json:"txn_time"`
}
