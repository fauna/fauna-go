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
		return
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
	cli.logger.LogResponse(cli.ctx, bytesOut, httpRes)

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

func parseQueryResponse(httpRes *http.Response) (qRes *queryResponse, err error) {
	var bytesIn []byte
	if bytesIn, err = io.ReadAll(httpRes.Body); err != nil {
		err = fmt.Errorf("failed to read response body: %w", err)
		return
	}

	if err = json.Unmarshal(bytesIn, &qRes); err != nil {
		err = fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return
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
		return
	}

	var qRes *queryResponse
	if qRes, err = parseQueryResponse(httpRes); err != nil {
		return
	}
	cli.logger.LogResponse(cli.ctx, bytesOut, httpRes)

	cli.lastTxnTime.sync(qRes.TxnTime)
	qRes.Header = httpRes.Header

	if err = getErrFauna(httpRes.StatusCode, qRes, attempts); err != nil {
		return
	}

	var data any
	if data, err = decode(qRes.Data); err != nil {
		err = fmt.Errorf("failed to decode data: %w", err)
		return
	}

	qSus = &QuerySuccess{
		QueryInfo:  newQueryInfo(qRes),
		Data:       data,
		StaticType: qRes.StaticType,
	}
	qSus.Stats.Attempts = attempts
	return
}

type streamRequest struct {
	apiRequest
	Stream  EventSource
	StartTS int64
	Cursor  string
}

func (streamReq *streamRequest) do(cli *Client) (bytes io.ReadCloser, err error) {
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
		return
	}
	cli.logger.LogResponse(cli.ctx, bytesOut, httpRes)

	if httpRes.StatusCode != http.StatusOK {
		var qRes *queryResponse
		if qRes, err = parseQueryResponse(httpRes); err == nil {
			if err = getErrFauna(httpRes.StatusCode, qRes, attempts); err == nil {
				err = fmt.Errorf("unknown error for http status: %d", httpRes.StatusCode)
			}
		}
		return
	}

	bytes = httpRes.Body
	return
}

type feedRequest struct {
	apiRequest
	Stream  EventSource
	StartTS int64
	Cursor  string
}

func (feedReq *feedRequest) do(cli *Client) (io.ReadCloser, error) {
	bytesOut, marshalErr := marshal(feedReq)
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal request failed: %w", marshalErr)
	}

	changeFeedURL, parseURLErr := cli.parseFeedURL()
	if parseURLErr != nil {
		return nil, fmt.Errorf("parse url failed: %w", parseURLErr)
	}

	attempts, httpRes, postErr := feedReq.post(cli, changeFeedURL, bytesOut)
	if postErr != nil {
		return nil, fmt.Errorf("post request failed: %w", postErr)
	}

	if httpRes.StatusCode != http.StatusOK {
		qRes, err := parseQueryResponse(httpRes)
		if err == nil {
			if err = getErrFauna(httpRes.StatusCode, qRes, attempts); err == nil {
				err = fmt.Errorf("unknown error for http status: %d", httpRes.StatusCode)
			}
		}

		return nil, err
	}

	return httpRes.Body, nil
}
