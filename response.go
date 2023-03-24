package fauna

type Stats struct {
	ComputeOps        int `json:"compute_ops"`
	ReadOps           int `json:"read_ops"`
	WriteOps          int `json:"write_ops"`
	QueryTimeMs       int `json:"query_time_ms"`
	ContentionRetries int `json:"contention_retries"`
	StorageBytesRead  int `json:"storage_bytes_read"`
	StorageBytesWrite int `json:"storage_bytes_write"`
}

type QueryInfo struct {
	TxnTime   int64
	Summary   string
	QueryTags map[string]string
	Stats     *Stats
}

func newQueryInfo(res *queryResponse) *QueryInfo {
	return &QueryInfo{
		TxnTime:   res.TxnTime,
		Summary:   res.Summary,
		QueryTags: res.QueryTags(),
		Stats:     res.Stats,
	}
}

type QuerySuccess struct {
	*QueryInfo
	Data       any
	StaticType string
}

func (r *QuerySuccess) Unmarshal(into any) error {
	return decodeInto(r.Data, into)
}
