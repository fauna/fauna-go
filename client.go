// Package fauna provides a driver for Fauna FQL X
package fauna

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fauna/fauna-go/internal/fingerprinting"
)

//go:embed version
var driverVersion string

const (
	// EndpointDefault constant for Fauna Production endpoint
	EndpointDefault = "https://db.fauna.com"
	// EndpointLocal constant for local (Docker) endpoint
	EndpointLocal = "http://localhost:8443"

	// EnvFaunaEndpoint environment variable for Fauna Client HTTP endpoint
	EnvFaunaEndpoint = "FAUNA_ENDPOINT"
	// EnvFaunaSecret environment variable for Fauna Client authentication
	EnvFaunaSecret = "FAUNA_SECRET"

	// Headers consumers might want to use

	HeaderLastTxnTs            = "X-Last-Txn-Ts"
	HeaderLinearized           = "X-Linearized"
	HeaderMaxContentionRetries = "X-Max-Contention-Retries"
	HeaderTags                 = "X-Query-Tags"
	HeaderQueryTimeoutMs       = "X-Query-Timeout-Ms"
	HeaderTraceparent          = "Traceparent"
	HeaderTypecheck            = "X-Typecheck"

	// Headers just used internally

	headerAuthorization = "Authorization"
	headerContentType   = "Content-Type"
	headerDriver        = "X-Driver"
	headerDriverEnv     = "X-Driver-Env"
	headerFormat        = "X-Format"

	retryMaxAttemptsDefault = 3
	retryMaxBackoffDefault  = time.Second * 20
)

// Client is the Fauna Client.
type Client struct {
	url                 string
	secret              string
	headers             map[string]string
	lastTxnTime         txnTime
	typeCheckingEnabled bool

	http *http.Client
	ctx  context.Context

	maxAttempts int
	maxBackoff  time.Duration

	// lazily cached URLs
	queryURL, streamURL *url.URL
}

// NewDefaultClient initialize a [fauna.Client] with recommend default settings
func NewDefaultClient() (*Client, error) {
	var secret string
	if val, found := os.LookupEnv(EnvFaunaSecret); !found {
		return nil, fmt.Errorf("unable to load key from environment variable '%s'", EnvFaunaSecret)
	} else {
		secret = val
	}

	url, urlFound := os.LookupEnv(EnvFaunaEndpoint)
	if !urlFound {
		url = EndpointDefault
	}

	return NewClient(
		secret,
		DefaultTimeouts(),
		URL(url),
	), nil
}

type Timeouts struct {
	// The timeout of each query. This controls the maximum amount of time Fauna will
	// execute your query before marking it failed.
	QueryTimeout time.Duration

	// Time beyond `QueryTimeout` at which the client will abort a request if it has not received a response.
	// The default is 5s, which should account for network latency for most clients. The value must be greater
	// than zero. The closer to zero the value is, the more likely the client is to abort the request before the
	// server can report a legitimate response or error.
	ClientBufferTimeout time.Duration

	// ConnectionTimeout amount of time to wait for the connection to complete.
	ConnectionTimeout time.Duration

	// IdleConnectionTimeout is the maximum amount of time an idle (keep-alive) connection will
	// remain idle before closing itself.
	IdleConnectionTimeout time.Duration
}

// DefaultTimeouts suggested timeouts for the default [fauna.Client]
func DefaultTimeouts() Timeouts {
	return Timeouts{
		QueryTimeout:          time.Second * 5,
		ClientBufferTimeout:   time.Second * 5,
		ConnectionTimeout:     time.Second * 5,
		IdleConnectionTimeout: time.Second * 5,
	}
}

// NewClient initialize a new [fauna.Client] with custom settings
func NewClient(secret string, timeouts Timeouts, configFns ...ClientConfigFn) *Client {
	dialer := net.Dialer{
		Timeout: timeouts.ConnectionTimeout,
	}

	// NOTE: prefer a response header timeout instead of a client timeout so
	// that the client don't stop reading a http body that was produced by
	// Fauna. On the query interface, an HTTP body is sent as a single http
	// message. On the streaming interface, HTTP chunks are sent on every event.
	// Therefore, it's in the driver's best interest to continue reading the
	// HTTP body once the headers appear.
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialer.DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          20,
			IdleConnTimeout:       timeouts.IdleConnectionTimeout,
			ResponseHeaderTimeout: timeouts.QueryTimeout + timeouts.ClientBufferTimeout,
		},
	}

	defaultHeaders := map[string]string{
		headerContentType: "application/json; charset=utf-8",
		headerDriver:      "go",
		headerDriverEnv: fmt.Sprintf(
			"driver=go-%s; runtime=%s env=%s; os=%s",
			strings.TrimSpace(driverVersion),
			fingerprinting.Version(),
			fingerprinting.Environment(),
			fingerprinting.EnvironmentOS(),
		),
		headerFormat: "tagged",
	}

	if timeouts.QueryTimeout > 0 {
		defaultHeaders[HeaderQueryTimeoutMs] = fmt.Sprintf("%v", timeouts.QueryTimeout.Milliseconds())
	}

	client := &Client{
		ctx:                 context.TODO(),
		secret:              secret,
		http:                httpClient,
		url:                 EndpointDefault,
		headers:             defaultHeaders,
		lastTxnTime:         txnTime{},
		typeCheckingEnabled: false,
		maxAttempts:         retryMaxAttemptsDefault,
		maxBackoff:          retryMaxBackoffDefault,
	}

	// set options to override defaults
	for _, configFn := range configFns {
		configFn(client)
	}

	return client
}

func (c *Client) parseQueryURL() (url *url.URL, err error) {
	if c.queryURL != nil {
		url = c.queryURL
	} else if url, err = url.Parse(c.url); err == nil {
		url = url.JoinPath("query", "1")
		c.queryURL = url
	}
	return
}

func (c *Client) parseStreamURL() (url *url.URL, err error) {
	if c.streamURL != nil {
		url = c.streamURL
	} else if url, err = url.Parse(c.url); err == nil {
		url = url.JoinPath("stream", "1")
		c.streamURL = url
	}
	return
}

func (c *Client) doWithRetry(req *http.Request) (attempts int, r *http.Response, err error) {
	req2 := req.Clone(req.Context())
	body, rerr := io.ReadAll(req.Body)
	if rerr != nil {
		return attempts, r, rerr
	}

	cerr := req.Body.Close()
	if cerr != nil {
		return attempts, r, cerr
	}

	for {
		shouldRetry := false

		// Ensure we have a fresh body for the request
		req2.Body = io.NopCloser(bytes.NewReader(body))
		r, err = c.http.Do(req2)
		attempts++
		if err != nil {
			return
		}

		if r != nil {
			switch r.StatusCode {
			case http.StatusTooManyRequests:
				shouldRetry = true
			}
		}

		if attempts >= c.maxAttempts || !shouldRetry {
			return attempts, r, err
		}

		// We're going to retry, so drain the response
		if r != nil {
			err = c.drainResponse(r.Body)
			if err != nil {
				return
			}
		}

		timer := time.NewTimer(c.backoff(attempts))
		select {
		case <-req.Context().Done():
			timer.Stop()
			return attempts, r, req.Context().Err()
		case <-timer.C:
		}
	}
}

func (c *Client) drainResponse(body io.ReadCloser) (err error) {
	defer body.Close()
	_, err = io.Copy(io.Discard, io.LimitReader(body, 4096))
	return
}

func (c *Client) backoff(attempt int) (sleep time.Duration) {
	jitter := rand.Float64()
	mult := math.Pow(2, float64(attempt)) * jitter
	sleep = time.Duration(mult) * time.Second

	if sleep > c.maxBackoff {
		sleep = c.maxBackoff
	}
	return
}

// Query invoke fql optionally set multiple [QueryOptFn]
func (c *Client) Query(fql *Query, opts ...QueryOptFn) (*QuerySuccess, error) {
	req := &queryRequest{
		apiRequest: apiRequest{
			Context: c.ctx,
			Headers: c.headers,
		},
		Query: fql,
	}

	for _, queryOptionFn := range opts {
		queryOptionFn(req)
	}

	return req.do(c)
}

// Paginate invoke fql with pagination optionally set multiple [QueryOptFn]
func (c *Client) Paginate(fql *Query, opts ...QueryOptFn) *QueryIterator {
	return &QueryIterator{
		client: c,
		fql:    fql,
		opts:   opts,
	}
}

// Subscribe initiates a stream subscription for the given stream value.
func (c *Client) Subscribe(stream Stream) (*Subscription, error) {
	streamReq := streamRequest{
		apiRequest: apiRequest{c.ctx, c.headers},
		Stream:     stream,
	}

	if byteStream, err := streamReq.do(c); err == nil {
		sub := &Subscription{
			events:     make(chan *Event),
			byteStream: byteStream,
		}
		go sub.consume()
		return sub, nil
	} else {
		return nil, err
	}
}

// QueryIterator is a [fauna.Client] iterator for paginated queries
type QueryIterator struct {
	client *Client
	fql    *Query
	opts   []QueryOptFn
}

// Next returns the next page of results
func (q *QueryIterator) Next() (*Page, error) {
	res, queryErr := q.client.Query(q.fql, q.opts...)
	if queryErr != nil {
		return nil, queryErr
	}

	if page, ok := res.Data.(*Page); ok { // First page
		if pageErr := q.nextPage(page.After); pageErr != nil {
			return nil, pageErr
		}

		return page, nil
	}

	var page Page
	if results, isPage := res.Data.(map[string]any); isPage {
		if after, hasAfter := results["after"].(string); hasAfter {
			page = Page{After: after, Data: results["data"].([]any)}
		} else {
			page = Page{After: "", Data: results["data"].([]any)}
		}
	} else {
		page = Page{After: "", Data: []any{res.Data}}
	}

	if pageErr := q.nextPage(page.After); pageErr != nil {
		return nil, pageErr
	}

	return &page, nil
}

func (q *QueryIterator) nextPage(after string) error {
	if after == "" {
		q.fql = nil
		return nil
	}

	var fqlErr error
	q.fql, fqlErr = FQL(`Set.paginate(${after})`, map[string]any{"after": after})

	return fqlErr
}

// HasNext returns whether there is another page of results
func (q *QueryIterator) HasNext() bool {
	return q.fql != nil
}

// SetLastTxnTime update the last txn time for the [fauna.Client]
// This has no effect if earlier than stored timestamp.
//
// WARNING: This should be used only when coordinating timestamps across multiple clients.
// Moving the timestamp arbitrarily forward into the future will cause transactions to stall.
func (c *Client) SetLastTxnTime(txnTime time.Time) {
	c.lastTxnTime.Lock()
	defer c.lastTxnTime.Unlock()

	if val := txnTime.UnixMicro(); val > c.lastTxnTime.Value {
		c.lastTxnTime.Value = val
	}
}

// GetLastTxnTime gets the last txn timestamp seen by the [fauna.Client]
func (c *Client) GetLastTxnTime() int64 {
	c.lastTxnTime.RLock()
	defer c.lastTxnTime.RUnlock()

	return c.lastTxnTime.Value
}

// String fulfil Stringify interface for the [fauna.Client]
// only returns the URL to prevent logging potentially sensitive headers.
func (c *Client) String() string {
	return c.url
}

func (c *Client) setHeader(key, val string) {
	c.headers[key] = val
}

type txnTime struct {
	sync.RWMutex

	Value int64
}

func (t *txnTime) string() string {
	t.RLock()
	defer t.RUnlock()

	if lastSeen := atomic.LoadInt64(&t.Value); lastSeen != 0 {
		return strconv.FormatInt(lastSeen, 10)
	}

	return ""
}

func (t *txnTime) sync(newTxnTime int64) {
	t.Lock()
	defer t.Unlock()

	for {
		oldTxnTime := atomic.LoadInt64(&t.Value)
		if oldTxnTime >= newTxnTime ||
			atomic.CompareAndSwapInt64(&t.Value, oldTxnTime, newTxnTime) {
			break
		}
	}
}
