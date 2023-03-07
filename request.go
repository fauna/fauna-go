package fauna

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
)

// QueryArgItem query args structure
type QueryArgItem struct {
	Key   string
	Value interface{}
}

// QueryArg create an [QueryArgItem]
func QueryArg(key string, value interface{}) QueryArgItem {
	return QueryArgItem{
		Key:   key,
		Value: value,
	}
}

// QueryArgs map from [QueryArgItem]
type QueryArgs map[string]interface{}

// QueryArguments convenience method to structure [QueryArgs]
func QueryArguments(args ...QueryArgItem) QueryArgs {
	out := map[string]interface{}{}
	for n := range args {
		arg := args[n]
		out[arg.Key] = arg.Value
	}

	return out
}

type fqlRequest struct {
	Context   context.Context        `json:"-"`
	Headers   map[string]string      `json:"-"`
	Query     string                 `json:"query"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

func (c *Client) do(request *fqlRequest) (*Response, error) {
	bytesOut, bytesErr := json.Marshal(request)
	if bytesErr != nil {
		return nil, fmt.Errorf("marshal request failed: %w", bytesErr)
	}

	reqURL, urlErr := url.Parse(c.url)
	if urlErr != nil {
		return nil, urlErr
	}

	reqURL.Path = path.Join(reqURL.Path, "/query/1")

	req, reqErr := http.NewRequestWithContext(request.Context, http.MethodPost, reqURL.String(), bytes.NewReader(bytesOut))
	if reqErr != nil {
		return nil, fmt.Errorf("failed to init request: %w", reqErr)
	}

	req.Header.Set(HeaderAuthorization, `Bearer `+c.secret)
	req.Header.Set(HeaderFormat, "simple")
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
