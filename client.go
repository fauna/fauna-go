// Package fauna HTTP client for fqlx
package fauna

import (
	"context"
	"crypto/tls"
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fauna/fauna-go/internal/fingerprinting"
	"golang.org/x/net/http2"
)

// DriverVersion semantic version of the driver
//
//go:embed version
var DriverVersion string

const (
	// EndpointProduction constant for Fauna Production endpoint
	EndpointProduction = "https://db.fauna.com"
	// EndpointPreview constant for Fauna Preview endpoint
	EndpointPreview = "https://db.fauna-preview.com"
	// EndpointLocal constant for local (Docker) endpoint
	EndpointLocal = "http://localhost:8443"

	// EnvFaunaEndpoint environment variable for Fauna Client HTTP endpoint
	EnvFaunaEndpoint = "FAUNA_ENDPOINT"
	// EnvFaunaSecret environment variable for Fauna Client authentication
	EnvFaunaSecret = "FAUNA_SECRET"

	// DefaultHttpReadIdleTimeout Fauna Client default HTTP read idle timeout
	DefaultHttpReadIdleTimeout = time.Minute * 3

	// Headers consumers might want to use

	HeaderLastTxnTs            = "X-Last-Txn-Ts"
	HeaderLinearized           = "X-Linearized"
	HeaderMaxContentionRetries = "X-Max-Contention-Retries"
	HeaderTags                 = "X-Query-Tags"
	HeaderTimeoutMs            = "X-Timeout-Ms"
	HeaderTraceparent          = "Traceparent"
	HeaderTypeChecking         = "X-Type-Checking"

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
		url = EndpointProduction
	}

	return NewClient(
		secret,
		URL(url),
		HTTPClient(defaultHTTPClient(url == EndpointLocal)),
		Context(context.TODO()),
	), nil
}

// NewClient initialize a new [fauna.Client] with custom settings
func NewClient(secret string, configFns ...ClientConfigFn) *Client {
	client := &Client{
		ctx:    context.TODO(),
		secret: secret,
		http:   defaultHTTPClient(false),
		url:    EndpointProduction,
		headers: map[string]string{
			headerContentType:   "application/json; charset=utf-8",
			headerDriver:        "go",
			headerDriverVersion: DriverVersion,
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

// Query invoke fql with args and map to the provided obj, optionally set multiple [QueryOptFn]
func (c *Client) Query(fql string, args QueryArgs, obj any, opts ...QueryOptFn) (*Response, error) {
	req := &fqlRequest{
		Context:   c.ctx,
		Query:     fql,
		Arguments: args,
		Headers:   c.headers,
	}

	for _, queryOptionFn := range opts {
		queryOptionFn(req)
	}

	res, err := c.do(req)
	if err != nil {
		return res, err
	}

	// we only need to unmarshal if the consumer provided an object
	if obj != nil {
		if unmarshalErr := unmarshal(res.Data, obj); unmarshalErr != nil {
			return res, fmt.Errorf("failed to unmarshal object [%v] from result: %v\nerror: %w", obj, string(res.Data), unmarshalErr)
		}
	}

	return res, nil
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

func defaultHTTPClient(allowHTTP bool) *http.Client {
	return &http.Client{
		Transport: &http2.Transport{
			DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
			AllowHTTP:        allowHTTP,
			ReadIdleTimeout:  DefaultHttpReadIdleTimeout,
			PingTimeout:      time.Second * 3,
			WriteByteTimeout: time.Second * 5,
		},
	}
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
