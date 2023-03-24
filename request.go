package fauna

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type fqlRequest struct {
	Context   context.Context
	Headers   map[string]string
	Query     any            `fauna:"query"`
	Arguments map[string]any `fauna:"arguments"`
}

func (c *Client) do(request *fqlRequest) (*Response, error) {
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
		return nil, NetworkError(fmt.Errorf("network error: %w", doErr))
	}

	defer func() {
		_ = req.Body.Close()
	}()

	var response Response
	response.Raw = r

	bin, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read response body: %w", readErr)
	}

	response.Bytes = bin

	if unmarshalErr := json.Unmarshal(bin, &response); unmarshalErr != nil {
		return &response, fmt.Errorf("failed to umarmshal response: %w", unmarshalErr)
	}

	c.lastTxnTime.sync(response.TxnTime)

	if serviceErr := GetServiceError(r.StatusCode, response.Error, response.Summary); serviceErr != nil {
		return &response, serviceErr
	}

	return &response, nil
}
