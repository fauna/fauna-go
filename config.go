package fauna

import (
	"context"
	"fmt"
	"net/http"
	"time"
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
		for k, v := range headers {
			c.headers[k] = v
		}
	}
}

// TypeChecking toggle if fauna.Client enforces type checking
func TypeChecking(enabled bool) ClientConfigFn {
	return func(c *Client) {
		c.typeCheckingEnabled = enabled
	}
}

// LastTxnTime toggle if fauna.Client records the last transaction time
func LastTxnTime(enabled bool) ClientConfigFn {
	return func(c *Client) {
		c.txnTimeEnabled = enabled
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

// SetHeader update fauna.Client header
func (c *Client) SetHeader(key, val string) {
	c.headers[key] = val
}

// SetTypeChecking update fauna.Client type checking setting
func (c *Client) SetTypeChecking(enabled bool) {
	c.typeCheckingEnabled = enabled
}

type QueryOptFn func(req *fqlRequest)

// QueryContext set the context.Context for the Query
func QueryContext(ctx context.Context) QueryOptFn {
	return func(req *fqlRequest) {
		req.Context = ctx
	}
}

// QueryTxnTime toggle if fauna.Client records the last transaction for the Query
func QueryTxnTime(enabled bool) QueryOptFn {
	return func(req *fqlRequest) {
		req.TxnTimeEnabled = enabled
	}
}

// QueryTypeChecking toggle if fauna.Client uses type checking
func QueryTypeChecking(enabled bool) QueryOptFn {
	return func(req *fqlRequest) {
		req.Headers[HeaderTypeChecking] = fmt.Sprintf("%v", enabled)
	}
}

// QueryTimeout set the query timeout
func QueryTimeout(dur time.Duration) QueryOptFn {
	return func(req *fqlRequest) {
		req.Headers[HeaderTypeChecking] = fmt.Sprintf("%f", dur.Seconds())
	}
}
