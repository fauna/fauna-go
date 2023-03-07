package fauna

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
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
	Context             context.Context        `json:"-"`
	Headers             map[string]string      `json:"-"`
	VerboseDebugEnabled bool                   `json:"-"`
	Query               string                 `json:"query"`
	Arguments           map[string]interface{} `json:"arguments,omitempty"`
}

func (c *Client) do(request *fqlRequest) (*Response, error) {
	bytesOut, bytesErr := json.Marshal(request)
	if bytesErr != nil {
		return nil, fmt.Errorf("marshal request failed: %w", bytesErr)
	}

	req, reqErr := http.NewRequestWithContext(request.Context, http.MethodPost, c.url, bytes.NewReader(bytesOut))
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

	if request.VerboseDebugEnabled {
		reqDump, dumpErr := httputil.DumpRequestOut(req, true)
		if dumpErr != nil {
			c.log.Printf("Failed to dump request: %s", dumpErr.Error())
		} else {
			c.log.Printf("REQUEST:\n%s\n-------", string(reqDump))
		}
	}

	r, doErr := c.http.Do(req)

	if request.VerboseDebugEnabled {
		respDump, dumpErr := httputil.DumpResponse(r, true)
		if dumpErr != nil {
			c.log.Printf("Failed to dump response: %s", dumpErr.Error())
		} else {
			c.log.Printf("RESPONSE:\n%s\n-------", string(respDump))
		}
	}

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
		c.log.Printf("[ERROR] %d - %v - %v\n%s", r.StatusCode, response.Error, response.Summary, response.Bytes)
		return &response, serviceErr
	}

	return &response, nil
}
