// Package fauna HTTP client for fqlx
package fauna

import (
	"context"
	"crypto/tls"
	_ "embed"
	"fmt"
	"log"
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
	// EnvFaunaTimeout environment variable for Fauna Client Read-Idle Timeout
	EnvFaunaTimeout = "FAUNA_TIMEOUT"
	// EnvFaunaTypeCheckEnabled environment variable for Fauna Client TypeChecking
	EnvFaunaTypeCheckEnabled = "FAUNA_TYPE_CHECK_ENABLED"

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

	readIdleTimeout := DefaultHttpReadIdleTimeout
	if val, found := os.LookupEnv(EnvFaunaTimeout); found {
		if timeoutFromEnv, err := time.ParseDuration(val); err != nil {
			log.Default().Printf("[WARNING] using default timeout - failed to parse timeout [%s]", err.Error())
		} else {
			if timeoutFromEnv.Seconds() <= 0 {
				log.Default().Printf("[WARNING] using default timeout - value must be greater than 0")
			} else {
				readIdleTimeout = timeoutFromEnv
			}
		}
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
				ReadIdleTimeout:  readIdleTimeout,
				PingTimeout:      time.Second * 3,
				WriteByteTimeout: time.Second * 5,
			},
		}),
		Context(context.TODO()),
	), nil
}

// NewClient initialize a new [fauna.Client] with custom settings
func NewClient(secret string, configFns ...ClientConfigFn) *Client {
	typeCheckEnabled := isEnabled(EnvFaunaTypeCheckEnabled, true)

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
		typeCheckingEnabled: typeCheckEnabled,
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
func (c *Client) SetLastTxnTime(txnTime time.Time) error {
	c.lastTxnTime.Lock()
	defer c.lastTxnTime.Unlock()

	val := txnTime.UnixMicro()
	if val < c.lastTxnTime.Value {
		return fmt.Errorf("unable to set last transaction time less than previously known value:\n\tcurrent value: %d\n\tattempted value: %d", c.lastTxnTime.Value, val)
	}

	c.lastTxnTime.Value = val

	return nil
}

// GetLastTxnTime gets the freshest timestamp reported to this client.
func (c *Client) GetLastTxnTime() int64 {
	c.lastTxnTime.RLock()
	defer c.lastTxnTime.RUnlock()

	return c.lastTxnTime.Value
}

// String fulfil Stringify interface for the [fauna.Client]
// only returns the URL to prevent logging potentially sensitive Headers.
func (c *Client) String() string {
	return c.url
}

func isEnabled(envVar string, defaultValue bool) bool {
	if val, found := os.LookupEnv(envVar); found {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal
		}
	}

	return defaultValue
}
