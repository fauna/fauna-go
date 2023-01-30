// Package fauna HTTP client for fqlx
package fauna

import (
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
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
	EndpointProduction = "https://db.fauna.com/query/1"
	// EndpointPreview constant for Fauna Preview endpoint
	EndpointPreview = "https://db.fauna-preview.com/query/1"
	// EndpointLocal constant for local (Docker) endpoint
	EndpointLocal = "http://localhost:8443/query/1"

	// EnvFaunaEndpoint environment variable for Fauna Client HTTP endpoint
	EnvFaunaEndpoint = "FAUNA_ENDPOINT"
	// EnvFaunaSecret environment variable for Fauna Client authentication
	EnvFaunaSecret = "FAUNA_SECRET"
	// EnvFaunaTimeout environment variable for Fauna Client Read-Idle Timeout
	EnvFaunaTimeout = "FAUNA_TIMEOUT"
	// EnvFaunaTypeCheckEnabled environment variable for Fauna Client TypeChecking
	EnvFaunaTypeCheckEnabled = "FAUNA_TYPE_CHECK_ENABLED"
	// EnvFaunaTrackTxnTimeEnabled environment variable for Fauna Client tracks Transaction time
	EnvFaunaTrackTxnTimeEnabled = "FAUNA_TRACK_TXN_TIME_ENABLED"

	EnvFaunaVerboseDebugEnabled = "FAUNA_VERBOSE_DEBUG_ENABLED"

	// DefaultHttpReadIdleTimeout Fauna Client default HTTP read idle timeout
	DefaultHttpReadIdleTimeout = time.Minute * 3

	// Reuest/response Headers
	HeaderContentType = "Content-Type"
	HeaderTxnTime     = "X-Txn-Time"

	// Request Headers
	HeaderAuthorization        = "Authorization"
	HeaderLastSeenTxn          = "X-Last-Seen-Txn"
	HeaderLinearized           = "X-Linearized"
	HeaderMaxContentionRetries = "X-Max-Contention-Retries"
	HeaderTimeoutMs            = "X-Timeout-Ms"
	HeaderTypeChecking         = "X-Fauna-Type-Checking"

	// Response Headers
	HeaderTraceparent       = "Traceparent"
	HeaderByteReadOps       = "X-Byte-Read-Ops"
	HeaderByteWriteOps      = "X-Byte-Write-Ops"
	HeaderComputeOps        = "X-Compute-Ops"
	HeaderFaunaBuild        = "X-Faunadb-Build"
	HeaderQueryBytesIn      = "X-Query-Bytes-In"
	HeaderQueryBytesOut     = "X-Query-Bytes-Out"
	HeaderQueryTime         = "X-Query-Time"
	HeaderReadOps           = "X-Read-Ops"
	HeaderStorageBytesRead  = "X-Storage-Bytes-Read"
	HeaderStorageBytesWrite = "X-Storage-Bytes-Write"
	HeaderTxnRetries        = "X-Txn-Retries"
	HeaderWriteOps          = "X-Write-Ops"
)

type txnTime struct {
	sync.RWMutex

	Enabled bool
	Value   int64
}

// Client is the Fauna Client.
type Client struct {
	url                 string
	secret              string
	headers             map[string]string
	lastTxnTime         txnTime
	typeCheckingEnabled bool
	verboseDebugEnabled bool

	http *http.Client
	log  *log.Logger
	ctx  context.Context

	// tags?
	// traceParent?
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
		timeoutFromEnv, err := time.ParseDuration(val)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to parse timeout, using default\n")
		} else {
			if timeoutFromEnv.Seconds() <= 0 {
				_, _ = fmt.Fprintf(os.Stderr, "timeout must be greater than 0, using default\n")
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
	verboseDebugEnabled := isEnabled(EnvFaunaVerboseDebugEnabled, false)

	client := &Client{
		ctx:    context.TODO(),
		log:    log.Default(),
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
		lastTxnTime: txnTime{
			Enabled: isEnabled(EnvFaunaTrackTxnTimeEnabled, true),
		},
		typeCheckingEnabled: typeCheckEnabled,
		verboseDebugEnabled: verboseDebugEnabled,
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
		Context:             c.ctx,
		Query:               fql,
		Arguments:           args,
		Headers:             c.headers,
		TxnTimeEnabled:      c.lastTxnTime.Enabled,
		VerboseDebugEnabled: c.verboseDebugEnabled,
	}

	for _, o := range opts {
		o(req)
	}
	res, err := c.do(req)
	if err != nil {
		return res, fmt.Errorf("request error: %w", err)
	}

	if obj != nil {
		unmarshalErr := json.Unmarshal(res.Data, obj)
		if unmarshalErr != nil {
			return res, fmt.Errorf("failed to unmarshal object [%v] from result: %v\nerror: %w", obj, res.Data, unmarshalErr)
		}
	}

	return res, nil
}

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

	if c.lastTxnTime.Enabled {
		return c.lastTxnTime.Value
	}

	return 0
}

func (c *Client) String() string {
	return fmt.Sprintf("%s", c.url)
}

func isEnabled(envVar string, defaultValue bool) bool {
	if val, found := os.LookupEnv(envVar); found {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal
		}
	}

	return defaultValue
}
