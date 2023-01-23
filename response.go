package fauna

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// Stats represents the metrics returned in a [Response]
type Stats struct {
	ReadOps    int           `json:"read_ops"`
	WriteOps   int           `json:"write_ops"`
	ComputeOps int           `json:"compute_ops"`
	QueryTime  time.Duration `json:"query_time"`
}

// The Response from a [Client.Query]
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

func (r Response) ByteReadOps() int {
	return intFromResponseHeader(r.Raw, HeaderByteReadOps)
}

func (r Response) ByteWriteOps() int {
	return intFromResponseHeader(r.Raw, HeaderByteWriteOps)
}

func (r Response) ComputeOps() int {
	return intFromResponseHeader(r.Raw, HeaderComputeOps)
}

func (r Response) FaunaBuild() string {
	return stringFromResponseHeader(r.Raw, HeaderFaunaBuild)
}

func (r Response) QueryTime() time.Duration {
	return time.Duration(int64(intFromResponseHeader(r.Raw, HeaderQueryTime)) * int64(time.Millisecond))
}

func (r Response) QueryBytesIn() int {
	return intFromResponseHeader(r.Raw, HeaderQueryBytesIn)
}

func (r Response) QueryBytesOut() int {
	return intFromResponseHeader(r.Raw, HeaderQueryBytesOut)
}

func (r Response) ReadOps() int {
	return intFromResponseHeader(r.Raw, HeaderReadOps)
}

func (r Response) StorageBytesRead() int {
	return intFromResponseHeader(r.Raw, HeaderStorageBytesRead)
}

func (r Response) StorageBytesWrite() int {
	return intFromResponseHeader(r.Raw, HeaderStorageBytesWrite)
}

func (r Response) Traceparent() string {
	return stringFromResponseHeader(r.Raw, HeaderTraceparent)
}

func (r Response) TxnRetries() int {
	return intFromResponseHeader(r.Raw, HeaderTxnRetries)
}

func (r Response) WriteOps() int {
	return intFromResponseHeader(r.Raw, HeaderWriteOps)
}

func stringFromResponseHeader(r *http.Response, key string) string {
	if r != nil {
		return r.Header.Get(key)
	}

	return ""
}

func intFromResponseHeader(r *http.Response, key string) int {
	if r != nil {
		val, err := strconv.Atoi(r.Header.Get(key))
		if err != nil {
			return 0
		}

		return val
	}

	return 0
}
