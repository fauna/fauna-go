package fauna

import (
	"bytes"
	"context"
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

	HeaderAuthorization        = "Authorization"
	HeaderTxnTime              = "X-Txn-Time"
	HeaderLastSeenTxn          = "X-Last-Seen-Txn"
	HeaderLinearized           = "X-Linearized"
	HeaderMaxContentionRetries = "X-Max-Contention-Retries"
	HeaderTimeoutMs            = "X-Timeout-Ms"
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

// Headers specify headers to on the fauna.Client
func Headers(headers map[string]string) ClientConfigFn {
	return func(c *Client) {
		if c.headers != nil {
			for k, v := range headers {
				c.headers[k] = v
			}
		} else {
			c.headers = headers
		}
	}
}

// Linearized set header on the fauna.Client
// A boolean. If true, unconditionally run the query as strictly serialized/linearized.
// This affects read-only transactions, as transactions which write will be strictly serialized.
func Linearized(enabled bool) ClientConfigFn {
	return func(c *Client) {
		c.SetHeader(HeaderLinearized, fmt.Sprintf("%v", enabled))
	}
}

// MaxContentionRetries set header on the fauna.Client
// An integer. The maximum number of times a transaction is retried due to OCC failure.
func MaxContentionRetries(i int) ClientConfigFn {
	return func(c *Client) {
		c.SetHeader(HeaderMaxContentionRetries, fmt.Sprintf("%v", i))
	}
}

// Timeout set header on the fauna.Client
func Timeout(d time.Duration) ClientConfigFn {
	return func(c *Client) {
		c.SetHeader(HeaderTimeoutMs, fmt.Sprintf("%v", d.Milliseconds()))
	}
}

// Context specify the context to be used for fauna.Client
func Context(ctx context.Context) ClientConfigFn {
	return func(c *Client) { c.ctx = ctx }
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
	ctx  context.Context

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
		Context(context.TODO()),
		Timeout(DefaultTimeout),
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
		ctx:    context.TODO(),
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

// SetHeader update fauna.Client header
func (c *Client) SetHeader(key, val string) {
	if c.headers != nil {
		c.headers[key] = val
	} else {
		c.headers = map[string]string{
			key: val,
		}
	}
}

// SetTypeChecking update fauna.Client type checking setting
func (c *Client) SetTypeChecking(enabled bool) {
	c.typeCheckingEnabled = enabled
}

// Query invoke fql with args and map to the provided obj
func (c *Client) Query(fql string, args map[string]interface{}, obj any) (*Response, error) {
	return c.query(c.ctx, fql, args, obj, c.typeCheckingEnabled)
}

func (c *Client) QueryWithOptions(fql string, args map[string]interface{}, obj any, opts ...ClientConfigFn) (*Response, error) {
	tempClient := *c
	for _, o := range opts {
		o(&tempClient)
	}

	return tempClient.query(tempClient.ctx, fql, args, obj, tempClient.typeCheckingEnabled)
}

type fqlRequest struct {
	Query     string                 `json:"query"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	TypeCheck bool                   `json:"typecheck"`
}

func (c *Client) query(ctx context.Context, fql string, args map[string]interface{}, obj any, typeChecking bool) (*Response, error) {
	res, err := c.do(ctx, &fqlRequest{
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

func (c *Client) do(ctx context.Context, request *fqlRequest) (*Response, error) {
	bytesOut, bytesErr := json.Marshal(request)
	if bytesErr != nil {
		return nil, bytesErr
	}

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(bytesOut))
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
