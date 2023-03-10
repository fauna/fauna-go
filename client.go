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

	// Request/response Headers

	HeaderContentType = "Content-Type"
	HeaderLastTxnTs   = "X-Last-Txn-Ts"

	// Request Headers

	HeaderAuthorization        = "Authorization"
	HeaderFormat               = "X-Format"
	HeaderLinearized           = "X-Linearized"
	HeaderMaxContentionRetries = "X-Max-Contention-Retries"
	HeaderTags                 = "X-Tags"
	HeaderTimeoutMs            = "X-Timeout-Ms"
	HeaderTypeChecking         = "X-Type-Checking"

	// Response Headers

	HeaderTraceparent = "Traceparent"
	HeaderFaunaBuild  = "X-Faunadb-Build"
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
		HTTPClient(&http.Client{
			Transport: &http2.Transport{
				DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
					return net.Dial(network, addr)
				},
				AllowHTTP:        url == EndpointLocal,
				ReadIdleTimeout:  DefaultHttpReadIdleTimeout,
				PingTimeout:      time.Second * 3,
				WriteByteTimeout: time.Second * 5,
			},
		}),
		Context(context.TODO()),
	), nil
}

// NewClient initialize a new [fauna.Client] with custom settings
func NewClient(secret string, configFns ...ClientConfigFn) *Client {
	client := &Client{
		ctx:    context.TODO(),
		secret: secret,
		http:   http.DefaultClient,
		url:    EndpointProduction,
		headers: map[string]string{
			HeaderContentType:          "application/json; charset=utf-8",
			"X-Fauna-Driver":           DriverVersion,
			"X-Runtime-Environment-OS": fingerprinting.EnvironmentOS(),
			"X-Runtime-Environment":    fingerprinting.Environment(),
			"X-Go-Version":             fingerprinting.Version(),
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

// Query invoke fql with args and map to the provided obj, optionally set [QueryOptFn]
func (c *Client) Query(fql string, args QueryArgs, obj interface{}, opts ...QueryOptFn) (*Response, error) {
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
		if unmarshalErr := decode(res.Data, obj); unmarshalErr != nil {
			return res, fmt.Errorf("failed to unmarshal object [%v] from result: %v\nerror: %w", obj, res.Data, unmarshalErr)
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
