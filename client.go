package fauna

import (
	"context"
	"crypto/tls"
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fauna/fauna-go/internal/runtime"
	"golang.org/x/net/http2"
)

//go:embed version
var DriverVersion string

const (
	EndpointProduction = "https://db.fauna.com/query/1"
	EndpointPreview    = "https://db.fauna-preview.com/query/1"
	EndpointLocal      = "http://localhost:8443/query/1"

	EnvFaunaEndpoint         = "FAUNA_ENDPOINT"
	EnvFaunaSecret           = "FAUNA_SECRET"
	EnvFaunaTimeout          = "FAUNA_TIMEOUT"
	EnvFaunaTypeCheckEnabled = "FAUNA_TYPE_CHECK_ENABLED"

	// DefaultTimeout for both the http.Request and the HeaderTimeoutMs
	DefaultTimeout = time.Minute

	HeaderAuthorization        = "Authorization"
	HeaderContentType          = "Content-Type"
	HeaderTxnTime              = "X-Txn-Time"
	HeaderLastSeenTxn          = "X-Last-Seen-Txn"
	HeaderLinearized           = "X-Linearized"
	HeaderMaxContentionRetries = "X-Max-Contention-Retries"
	HeaderTimeoutMs            = "X-Timeout-Ms"
)

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
			"X-Runtime-Environment-OS": runtime.EnvironmentOS(),
			"X-Runtime-Environment":    runtime.Environment(),
			"X-Go-Version":             runtime.Version(),
		}),
		Context(context.TODO()),
		Timeout(clientTimeout),
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
		ctx:                 context.TODO(),
		secret:              secret,
		http:                http.DefaultClient,
		url:                 EndpointProduction,
		headers:             map[string]string{},
		typeCheckingEnabled: typeCheckEnabled,
		txnTimeEnabled:      true,
	}

	// set options to override defaults
	for _, configFn := range configFns {
		configFn(client)
	}

	return client
}

// Query invoke fql with args and map to the provided obj
func (c *Client) Query(fql string, args QueryArgs, obj any) (*Response, error) {
	return c.query(c.ctx, fql, args, obj, c.typeCheckingEnabled)
}

// QueryWithOptions invoke fql with request options
func (c *Client) QueryWithOptions(fql string, args QueryArgs, obj any, opts ...ClientConfigFn) (*Response, error) {
	tempClient := *c
	for _, o := range opts {
		o(&tempClient)
	}

	return tempClient.query(tempClient.ctx, fql, args, obj, tempClient.typeCheckingEnabled)
}

// GetLastTxnTime gets the freshest timestamp reported to this client.
func (c *Client) GetLastTxnTime() int64 {
	if c.txnTimeEnabled {
		return c.lastTxnTime
	}

	return 0
}
