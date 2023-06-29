// Package fauna provides a driver for Fauna FQL X
package fauna

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
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

	// DefaultHttpTimeout Fauna Client default HTTP timeout
	DefaultHttpTimeout = time.Minute * 3

	// Headers consumers might want to use

	HeaderLastTxnTs            = "X-Last-Txn-Ts"
	HeaderLinearized           = "X-Linearized"
	HeaderMaxContentionRetries = "X-Max-Contention-Retries"
	HeaderTags                 = "X-Query-Tags"
	HeaderTimeoutMs            = "X-Timeout-Ms"
	HeaderTraceparent          = "Traceparent"
	HeaderTypecheck            = "X-Typecheck"

	// Headers just used internally

	headerAuthorization = "Authorization"
	headerContentType   = "Content-Type"
	headerDriver        = "X-Driver"
	headerDriverVersion = "X-Driver-Version"
	headerRuntime       = "X-Runtime"
	headerFormat        = "X-Format"
	headerFaunaBuild    = "X-Faunadb-Build"
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
		URL(url),
	), nil
}

// NewClient initialize a new [fauna.Client] with custom settings
func NewClient(secret string, configFns ...ClientConfigFn) *Client {
	httpClient := http.DefaultClient
	httpClient.Timeout = DefaultHttpTimeout

	client := &Client{
		ctx:    context.TODO(),
		secret: secret,
		http:   httpClient,
		url:    EndpointDefault,
		headers: map[string]string{
			headerContentType:   "application/json; charset=utf-8",
			headerDriver:        "go",
			headerDriverVersion: strings.TrimSpace(driverVersion),
			headerFormat:        "tagged",
			headerRuntime: fmt.Sprintf(
				"env=%s; os=%s; go=%s",
				fingerprinting.Environment(),
				fingerprinting.EnvironmentOS(),
				fingerprinting.Version(),
			),
		},
		lastTxnTime:         txnTime{},
		typeCheckingEnabled: false,
	}

	// set options to override defaults
	for _, configFn := range configFns {
		configFn(client)
	}

	return client
}

// Query invoke fql optionally set multiple [QueryOptFn]
func (c *Client) Query(fql *Query, opts ...QueryOptFn) (*QuerySuccess, error) {
	req := &fqlRequest{
		Context: c.ctx,
		Query:   fql,
		Headers: c.headers,
	}

	for _, queryOptionFn := range opts {
		queryOptionFn(req)
	}

	return c.do(req)
}

// Paginate invoke fql with pagination optionally set multiple [QueryOptFn]
func (c *Client) Paginate(fql *Query, opts ...QueryOptFn) *QueryIterator {
	return &QueryIterator{
		client: c,
		fql:    fql,
		opts:   opts,
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
