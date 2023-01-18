// Package fauna HTTP client for fqlx
package fauna

import (
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fauna/fauna-go/internal/fingerprinting"
	"golang.org/x/net/http2"
)

// DriverVersion semantic version of the driver
//
//go:embed version
var DriverVersion string

const (
	EndpointProduction = "https://db.fauna.com/query/1"
	EndpointPreview    = "https://db.fauna-preview.com/query/1"
	EndpointLocal      = "http://localhost:8443/query/1"

	EnvFaunaEndpoint            = "FAUNA_ENDPOINT"
	EnvFaunaSecret              = "FAUNA_SECRET"
	EnvFaunaTimeout             = "FAUNA_TIMEOUT"
	EnvFaunaTypeCheckEnabled    = "FAUNA_TYPE_CHECK_ENABLED"
	EnvFaunaTrackTxnTimeEnabled = "FAUNA_TRACK_TXN_TIME_ENABLED"

	// DefaultTimeout for both the http.Request and the HeaderTimeoutMs
	DefaultTimeout = time.Minute

	HeaderAuthorization        = "Authorization"
	HeaderContentType          = "Content-Type"
	HeaderTxnTime              = "X-Txn-Time"
	HeaderLastSeenTxn          = "X-Last-Seen-Txn"
	HeaderLinearized           = "X-Linearized"
	HeaderMaxContentionRetries = "X-Max-Contention-Retries"
	HeaderTimeoutMs            = "X-Timeout-Ms"
	HeaderTypeChecking         = "X-Fauna-Type-Checking"
)

// Client is the Fauna Client.
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

// DefaultClient initialize a [fauna.Client] with recommend default settings
func DefaultClient() (*Client, error) {
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

	clientTimeout := DefaultTimeout
	if val, found := os.LookupEnv(EnvFaunaTimeout); found {
		timeoutFromEnv, err := time.ParseDuration(val)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to parse timeout, using default\n")
		} else {
			clientTimeout = timeoutFromEnv
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
				ReadIdleTimeout:  DefaultTimeout,
				PingTimeout:      time.Second * 3,
				WriteByteTimeout: time.Second * 5,
			},
		}),
		Headers(map[string]string{
			HeaderAuthorization:        fmt.Sprintf("Bearer %s", secret),
			HeaderContentType:          "application/json; charset=utf-8",
			"X-Fauna-Driver":           DriverVersion,
			"X-Runtime-Environment-OS": fingerprinting.EnvironmentOS(),
			"X-Runtime-Environment":    fingerprinting.Environment(),
			"X-Go-Version":             fingerprinting.Version(),
		}),
		Context(context.TODO()),
		Timeout(clientTimeout),
	), nil
}

// NewClient initialize a new [fauna.Client] with custom settings
func NewClient(secret string, configFns ...ClientConfigFn) *Client {
	// sensible default
	typeCheckEnabled := true
	if typeCheckEnabledVal, found := os.LookupEnv(EnvFaunaTypeCheckEnabled); found {
		// TRICKY: invert boolean check, we only want to disable if explicitly set to false
		typeCheckEnabled = !(strings.ToLower(typeCheckEnabledVal) == "false")
	}

	txnTimeEnabled := true
	if val, found := os.LookupEnv(EnvFaunaTrackTxnTimeEnabled); found {
		// TRICKY: invert boolean check, we only want to disable if explicitly set to false
		txnTimeEnabled = !(strings.ToLower(val) == "false")
	}

	client := &Client{
		ctx:                 context.TODO(),
		secret:              secret,
		http:                http.DefaultClient,
		url:                 EndpointProduction,
		headers:             map[string]string{},
		typeCheckingEnabled: typeCheckEnabled,
		txnTimeEnabled:      txnTimeEnabled,
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
		Context:        c.ctx,
		Query:          fql,
		Arguments:      args,
		Headers:        c.headers,
		TxnTimeEnabled: c.txnTimeEnabled,
	}

	for _, o := range opts {
		o(req)
	}
	res, err := c.do(req)
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

// GetLastTxnTime gets the freshest timestamp reported to this client.
func (c *Client) GetLastTxnTime() int64 {
	if c.txnTimeEnabled {
		return c.lastTxnTime
	}

	return 0
}
