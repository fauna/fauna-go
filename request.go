package fauna

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type fqlRequest struct {
	Context   context.Context
	Headers   map[string]string
	Query     any            `fauna:"query"`
	Arguments map[string]any `fauna:"arguments"`
}

type queryResponse struct {
	Header     http.Header
	Data       json.RawMessage `json:"data"`
	Error      *ErrFauna       `json:"error,omitempty"`
	Logging    []string        `json:"logging,omitempty"`
	StaticType string          `json:"static_type"`
	Stats      *Stats          `json:"stats,omitempty"`
	Summary    string          `json:"summary"`
	TxnTime    int64           `json:"txn_ts"`
	Tags       string          `json:"query_tags"`
}

func (r *queryResponse) queryTags() map[string]string {
	ret := map[string]string{}

	if r.Tags != "" {
		for _, tag := range strings.Split(r.Tags, `,`) {
			tokens := strings.Split(tag, `=`)
			ret[tokens[0]] = tokens[1]
		}
	}

	return ret
}

func (c *Client) do(request *fqlRequest) (*QuerySuccess, error) {
	bytesOut, bytesErr := marshal(request)
	if bytesErr != nil {
		return nil, fmt.Errorf("marshal request failed: %w", bytesErr)
	}

	reqURL, urlErr := url.Parse(c.url)
	if urlErr != nil {
		return nil, urlErr
	}

	if path, err := url.JoinPath(reqURL.Path, "query", "1"); err != nil {
		return nil, err
	} else {
		reqURL.Path = path
	}

	req, reqErr := http.NewRequestWithContext(request.Context, http.MethodPost, reqURL.String(), bytes.NewReader(bytesOut))
	if reqErr != nil {
		return nil, fmt.Errorf("failed to init request: %w", reqErr)
	}

	req.Header.Set(headerAuthorization, `Bearer `+c.secret)
	if lastTxnTs := c.lastTxnTime.string(); lastTxnTs != "" {
		req.Header.Set(HeaderLastTxnTs, lastTxnTs)
	}

	for k, v := range request.Headers {
		req.Header.Set(k, v)
	}

	r, doErr := c.http.Do(req)
	if doErr != nil {
		return nil, ErrNetwork(fmt.Errorf("network error: %w", doErr))
	}

	var res queryResponse

	bin, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read response body: %w", readErr)
	}

	if unmarshalErr := json.Unmarshal(bin, &res); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to umarmshal response: %w", unmarshalErr)
	}

	c.lastTxnTime.sync(res.TxnTime)
	res.Header = r.Header

	if serviceErr := getErrFauna(r.StatusCode, &res); serviceErr != nil {
		return nil, serviceErr
	}

	data, decodeErr := decode(res.Data)
	if decodeErr != nil {
		return nil, fmt.Errorf("failed to decode data: %w", decodeErr)
	}

	ret := &QuerySuccess{
		QueryInfo:  newQueryInfo(&res),
		Data:       data,
		StaticType: res.StaticType,
	}

	return ret, nil
}
