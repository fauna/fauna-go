package fauna

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	EndpointProduction = "https://db.fauna.com/query/1"
	EndpointPreview    = "https://db.fauna-preview.com/query/1"
	EndpointLocal      = "http://localhost:8443/query/1"

	EnvFaunaEndpoint         = "FAUNA_ENDPOINT"
	EnvFaunaKey              = "FAUNA_KEY"
	EnvFaunaTypeCheckEnabled = "FAUNA_TYPE_CHECK_ENABLED"

	DefaultMaxConnections = 10
	DefaultTimeout        = time.Minute

	HeaderAuthorization = "Authorization"
	HeaderTxnTime       = "X-Txn-Time"
	HeaderLastSeenTxn   = "X-Last-Seen-Txn"
)

// ClientConfigFn configuration options for fauna.Client
type ClientConfigFn func(*Client)

// URL set the client URL
func URL(url string) ClientConfigFn {
	return func(c *Client) { c.url = url }
}

// HTTPClient set the http.Client for the fauna.Client
func HTTPClient(client *http.Client) ClientConfigFn {
	return func(c *Client) { c.http = client }
}

// Headers specify headers to be used on every http.Request
func Headers(headers map[string]string) ClientConfigFn {
	return func(c *Client) { c.headers = headers }
}

// Client type for
type Client struct {
	url                 string
	secret              string
	headers             map[string]string
	txnTimeEnabled      bool
	lastTxnTime         int64
	typeCheckingEnabled bool

	http *http.Client

	// maxRetries?
	// linearized?
	// tags?
	// traceParent?
}

// DefaultClient initialize fauna.Client with recommend settings
func DefaultClient() (*Client, error) {
	secret, found := os.LookupEnv(EnvFaunaKey)
	if !found {
		return nil, fmt.Errorf("unable to load key from environment variable '%s'", EnvFaunaKey)
	}

	url, urlFound := os.LookupEnv(EnvFaunaEndpoint)
	if !urlFound {
		url = EndpointProduction
	}

	return NewClient(
		secret,
		URL(url),
		HTTPClient(&http.Client{
			Transport: &http.Transport{
				MaxConnsPerHost:       DefaultMaxConnections,
				ResponseHeaderTimeout: DefaultTimeout,
			},
		}),
		Headers(map[string]string{
			HeaderAuthorization: fmt.Sprintf("Bearer %s", secret),
		}),
	), nil
}

// NewClient initialize a new fauna.Client with custom settings
func NewClient(secret string, configFns ...ClientConfigFn) *Client {
	// sensible default
	typeCheckEnabled := true
	if typeCheckEnabledVal, found := os.LookupEnv(EnvFaunaTypeCheckEnabled); found {
		// TRICKY: invert boolean check, we only want to disable if explicitly set to false
		typeCheckEnabled = !(strings.ToLower(typeCheckEnabledVal) == "false")
	}

	client := &Client{
		secret: secret,
		http:   http.DefaultClient,
		url:    EndpointProduction,
		headers: map[string]string{
			HeaderAuthorization: fmt.Sprintf("Bearer %s", secret),
		},
		typeCheckingEnabled: typeCheckEnabled,
	}

	// set options to override defaults
	for _, configFn := range configFns {
		configFn(client)
	}

	return client
}

// Query invoke fql with args and map to the provided obj
func (c *Client) Query(fql string, args map[string]interface{}, obj any) (*Response, error) {
	return c.query(fql, args, obj, c.typeCheckingEnabled)
}

// QueryPlain invoke `fql` without static checking enabled
func (c *Client) QueryPlain(fql string, args map[string]interface{}, obj any) (*Response, error) {
	return c.query(fql, args, obj, false)
}

type fqlRequest struct {
	Query     string                 `json:"query"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	TypeCheck bool                   `json:"typecheck"`
}

func (c *Client) query(fql string, args map[string]interface{}, obj any, typeChecking bool) (*Response, error) {
	res, err := c.do(&fqlRequest{
		Query:     fql,
		Arguments: args,
		TypeCheck: typeChecking,
	})
	if err != nil {
		return res, err
	}

	if obj != nil {
		unmarshalErr := json.Unmarshal(res.Data, obj)
		if unmarshalErr != nil {
			return res, unmarshalErr
		}
	}

	return res, nil
}

func (c *Client) do(request *fqlRequest) (*Response, error) {
	bytesOut, bytesErr := json.Marshal(request)
	if bytesErr != nil {
		return nil, bytesErr
	}

	req, reqErr := http.NewRequest(http.MethodPost, c.url, bytes.NewReader(bytesOut))
	if reqErr != nil {
		return nil, reqErr
	}

	req.Header.Set(HeaderAuthorization, fmt.Sprintf("Bearer %s", c.secret))
	for k, v := range c.headers {
		req.Header.Add(k, v)
	}

	if c.txnTimeEnabled {
		if lastSeen := atomic.LoadInt64(&c.lastTxnTime); lastSeen != 0 {
			req.Header.Set(HeaderLastSeenTxn, strconv.FormatInt(lastSeen, 10))
		}
	}

	r, doErr := c.http.Do(req)
	if doErr != nil {
		return nil, doErr
	}
	defer func() {
		_ = req.Body.Close()
	}()

	var response Response
	response.Raw = r

	if r.StatusCode >= http.StatusBadRequest {
		return &response, GetServiceError(r.StatusCode, response.Error)
	}

	bin, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		return nil, readErr
	}

	response.Bytes = bin

	if unmarshalErr := json.Unmarshal(bin, &response); unmarshalErr != nil {
		return &response, unmarshalErr
	}

	if txnTimeErr := c.storeLastTxnTime(r.Header); txnTimeErr != nil {
		return &response, txnTimeErr
	}

	if response.Error != nil {
		return &response, GetServiceError(r.StatusCode, response.Error)
	}

	return &response, nil
}

func (c *Client) storeLastTxnTime(header http.Header) error {
	if c.txnTimeEnabled {
		t, err := parseTxnTimeHeader(header)
		if err != nil {
			return err
		}
		c.syncLastTxnTime(t)
	}

	return nil
}

func (c *Client) syncLastTxnTime(newTxnTime int64) {
	if c.txnTimeEnabled {
		for {
			oldTxnTime := atomic.LoadInt64(&c.lastTxnTime)
			if oldTxnTime >= newTxnTime ||
				atomic.CompareAndSwapInt64(&c.lastTxnTime, oldTxnTime, newTxnTime) {
				break
			}
		}
	}
}

func parseTxnTimeHeader(header http.Header) (int64, error) {
	if h := header.Get(HeaderTxnTime); h != "" {
		return strconv.ParseInt(h, 10, 64)
	}

	return math.MinInt, nil
}
