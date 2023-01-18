package fauna

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync/atomic"
)

// QueryArgItem query args structure
type QueryArgItem struct {
	Key   string
	Value interface{}
}

// QueryArg create a QueryArgItem
func QueryArg(key string, value interface{}) QueryArgItem {
	return QueryArgItem{
		Key:   key,
		Value: value,
	}
}

// QueryArgs list of QueryArg items
type QueryArgs map[string]interface{}

// QueryArguments convenience method to structure QueryArgs
func QueryArguments(args ...QueryArgItem) QueryArgs {
	out := map[string]interface{}{}
	for n := range args {
		arg := args[n]
		out[arg.Key] = arg.Value
	}

	return out
}

type fqlRequest struct {
	Context        context.Context        `json:"-"`
	Headers        map[string]string      `json:"-"`
	TxnTimeEnabled bool                   `json:"-"`
	Query          string                 `json:"query"`
	Arguments      map[string]interface{} `json:"arguments,omitempty"`
	TypeCheck      bool                   `json:"typecheck"`
}

func (c *Client) do(request *fqlRequest) (*Response, error) {
	bytesOut, bytesErr := json.Marshal(request)
	if bytesErr != nil {
		return nil, bytesErr
	}

	req, reqErr := http.NewRequestWithContext(request.Context, http.MethodPost, c.url, bytes.NewReader(bytesOut))
	if reqErr != nil {
		return nil, fmt.Errorf("failed to init request: %w", reqErr)
	}

	req.Header.Set(HeaderAuthorization, `Bearer `+c.secret)
	for k, v := range request.Headers {
		req.Header.Set(k, v)
	}

	if request.TxnTimeEnabled {
		if lastSeen := atomic.LoadInt64(&c.lastTxnTime); lastSeen != 0 {
			req.Header.Set(HeaderLastSeenTxn, strconv.FormatInt(lastSeen, 10))
		}
	}

	r, doErr := c.http.Do(req)
	if doErr != nil {
		return nil, NetworkError(fmt.Errorf("request failed: %w", doErr))
	}

	defer func() {
		_ = req.Body.Close()
	}()

	var response Response
	response.Raw = r

	bin, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		return nil, readErr
	}

	response.Bytes = bin

	if unmarshalErr := json.Unmarshal(bin, &response); unmarshalErr != nil {
		return &response, fmt.Errorf("failed to umarmshal response: %w", unmarshalErr)
	}

	if request.TxnTimeEnabled {
		if txnTimeErr := c.storeLastTxnTime(r.Header); txnTimeErr != nil {
			return &response, fmt.Errorf("failed to parse transaction time: %w", txnTimeErr)
		}
	}

	if serviceErr := GetServiceError(r.StatusCode, response.Error); serviceErr != nil {
		return &response, serviceErr
	}

	return &response, nil
}

func (c *Client) storeLastTxnTime(header http.Header) error {
	t, err := parseTxnTimeHeader(header)
	if err != nil {
		return fmt.Errorf("failed to parse tranaction time: %w", err)
	}
	c.syncLastTxnTime(t)

	return nil
}

func (c *Client) syncLastTxnTime(newTxnTime int64) {
	if !c.txnTimeEnabled {
		return
	}

	for {
		oldTxnTime := atomic.LoadInt64(&c.lastTxnTime)
		if oldTxnTime >= newTxnTime ||
			atomic.CompareAndSwapInt64(&c.lastTxnTime, oldTxnTime, newTxnTime) {
			break
		}
	}
}

func parseTxnTimeHeader(header http.Header) (int64, error) {
	if h := header.Get(HeaderTxnTime); h != "" {
		return strconv.ParseInt(h, 10, 64)
	}

	return 0, nil
}
