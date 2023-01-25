package fauna

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"
	"sync/atomic"
)

// Format type cast the fauna format constants
type Format string

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
	TxnTimeEnabled      bool                   `json:"-"`
	VerboseDebugEnabled bool                   `json:"-"`
	Query               string                 `json:"query"`
	Arguments           map[string]interface{} `json:"arguments,omitempty"`
	TypeCheck           bool                   `json:"typecheck"`
	Format              Format                 `json:"format"`
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
	for k, v := range request.Headers {
		req.Header.Set(k, v)
	}

	if request.TxnTimeEnabled {
		c.lastTxnTime.RLock()
		if lastSeen := atomic.LoadInt64(&c.lastTxnTime.Value); lastSeen != 0 {
			req.Header.Set(HeaderLastSeenTxn, strconv.FormatInt(lastSeen, 10))
		}
		c.lastTxnTime.RUnlock()
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

	if request.TxnTimeEnabled {
		if txnTimeErr := c.storeLastTxnTime(r.Header); txnTimeErr != nil {
			return &response, fmt.Errorf("failed to parse transaction time: %w", txnTimeErr)
		}
	}

	if serviceErr := GetServiceError(r.StatusCode, response.Error, response.Summary); serviceErr != nil {
		c.log.Printf("[ERROR] %d - %v - %v\n%s", r.StatusCode, response.Error, response.Summary, response.Bytes)
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
	if !c.lastTxnTime.Enabled {
		return
	}

	c.lastTxnTime.Lock()
	defer c.lastTxnTime.Unlock()

	for {
		oldTxnTime := atomic.LoadInt64(&c.lastTxnTime.Value)
		if oldTxnTime >= newTxnTime ||
			atomic.CompareAndSwapInt64(&c.lastTxnTime.Value, oldTxnTime, newTxnTime) {
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
