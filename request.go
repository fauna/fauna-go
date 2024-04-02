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

type apiRequest struct {
	Context context.Context
	Headers map[string]string
}

func (apiReq *apiRequest) post(cli *Client, url *url.URL, bytesOut []byte) (attempts int, httpRes *http.Response, err error) {
	var httpReq *http.Request
	if httpReq, err = http.NewRequestWithContext(
		apiReq.Context,
		http.MethodPost,
		url.String(),
		bytes.NewReader(bytesOut),
	); err != nil {
		err = fmt.Errorf("failed to init request: %w", err)
	}

	httpReq.Header.Set(headerAuthorization, `Bearer `+cli.secret)
	if lastTxnTs := cli.lastTxnTime.string(); lastTxnTs != "" {
		httpReq.Header.Set(HeaderLastTxnTs, lastTxnTs)
	}

	for k, v := range apiReq.Headers {
		httpReq.Header.Set(k, v)
	}

	if attempts, httpRes, err = cli.doWithRetry(httpReq); err != nil {
		err = ErrNetwork(fmt.Errorf("network error: %w", err))
	}
	return
}

type queryRequest struct {
	apiRequest
	Query     any
	Arguments map[string]any
}

type queryResponse struct {
	Header        http.Header
	Data          json.RawMessage `json:"data"`
	Error         *ErrFauna       `json:"error,omitempty"`
	Logging       []string        `json:"logging,omitempty"`
	SchemaVersion int64           `json:"schema_version"`
	StaticType    string          `json:"static_type"`
	Stats         *Stats          `json:"stats,omitempty"`
	Summary       string          `json:"summary"`
	TxnTime       int64           `json:"txn_ts"`
	Tags          string          `json:"query_tags"`
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

func (qReq *queryRequest) do(cli *Client) (qSus *QuerySuccess, err error) {
	var bytesOut []byte
	if bytesOut, err = marshal(qReq); err != nil {
		err = fmt.Errorf("marshal request failed: %w", err)
		return
	}

	var queryURL *url.URL
	if queryURL, err = cli.parseQueryURL(); err != nil {
		return
	}

	var (
		attempts int
		httpRes  *http.Response
	)
	if attempts, httpRes, err = qReq.post(cli, queryURL, bytesOut); err != nil {
		err = ErrNetwork(fmt.Errorf("network error: %w", err))
		return
	}

	var (
		qRes    queryResponse
		bytesIn []byte
	)

	if bytesIn, err = io.ReadAll(httpRes.Body); err != nil {
		err = fmt.Errorf("failed to read response body: %w", err)
		return
	}

	if err = json.Unmarshal(bytesIn, &qRes); err != nil {
		err = fmt.Errorf("failed to umarmshal response: %w", err)
		return
	}

	cli.lastTxnTime.sync(qRes.TxnTime)
	qRes.Header = httpRes.Header

	if err = getErrFauna(httpRes.StatusCode, &qRes, attempts); err != nil {
		return
	}

	var data any
	if data, err = decode(qRes.Data); err != nil {
		err = fmt.Errorf("failed to decode data: %w", err)
		return
	}

	qSus = &QuerySuccess{
		QueryInfo:  newQueryInfo(&qRes),
		Data:       data,
		StaticType: qRes.StaticType,
	}
	qSus.Stats.Attempts = attempts
	return
}

type streamRequest struct {
	apiRequest
	Token     string
	StartTime int64
}

func (streamReq *streamRequest) do(cli *Client) (stream *Stream, err error) {
	var bytesOut []byte
	if bytesOut, err = marshal(streamReq); err != nil {
		err = fmt.Errorf("marshal request failed: %w", err)
		return
	}

	var streamURL *url.URL
	if streamURL, err = cli.parseStreamURL(); err != nil {
		return
	}

	var (
		attempts int
		httpRes  *http.Response
	)
	if attempts, httpRes, err = streamReq.post(cli, streamURL, bytesOut); err != nil {
		err = fmt.Errorf("network error: %w", err)
		return
	}

	if httpRes.StatusCode != http.StatusOK {
		var bytes []byte
		if bytes, err = io.ReadAll(httpRes.Body); err != nil {
			err = fmt.Errorf("failed to read response body: %w", err)
			return
		}

		var qRes *queryResponse
		if err = json.Unmarshal(bytes, &qRes); err != nil {
			err = fmt.Errorf("failed to umarmshal response: %w", err)
			return
		}

		if err = getErrFauna(httpRes.StatusCode, qRes, attempts); err == nil {
			err = fmt.Errorf("unknown api error: %d", httpRes.StatusCode)
		}
		return
	}

	stream = &Stream{streamReq.Context, httpRes.Body, nil, nil}
	return
}
