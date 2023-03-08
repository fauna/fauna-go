package fauna

import (
	"net/http"
)

const (
	StatsComputeOps        = "compute_ops"
	StatsReadOps           = "read_ops"
	StatsWriteOps          = "write_ops"
	StatsQueryTimeMs       = "query_time_ms"
	StatsContentionRetries = "contention_retries"
	StatsStorageBytesRead  = "storage_bytes_read"
	StatsStorageBytesWrite = "storage_bytes_write"
)

// The Response from a [Client.Query]
type Response struct {
	Bytes      []byte
	Data       interface{}   `json:"data"`
	Error      *ServiceError `json:"error,omitempty"`
	Logging    []string      `json:"logging,omitempty"`
	Raw        *http.Response
	StaticType string         `json:"static_type"`
	Stats      map[string]int `json:"stats,omitempty"`
	Summary    string         `json:"summary"`
	TxnTime    int64          `json:"txn_ts"`
}

func (r Response) FaunaBuild() string {
	return stringFromResponseHeader(r.Raw, HeaderFaunaBuild)
}

func (r Response) Traceparent() string {
	return stringFromResponseHeader(r.Raw, HeaderTraceparent)
}

func stringFromResponseHeader(r *http.Response, key string) string {
	if r != nil {
		return r.Header.Get(key)
	}

	return ""
}
