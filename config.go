package fauna

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ClientConfigFn configuration options for the [fauna.Client]
type ClientConfigFn func(*Client)

// Context specify the context to be used for the [fauna.Client]
func Context(ctx context.Context) ClientConfigFn {
	return func(c *Client) { c.ctx = ctx }
}

// Logger specify the [log.Logger] for the [fauna.Client]
func Logger(log *log.Logger) ClientConfigFn {
	return func(c *Client) { c.log = log }
}

// HTTPClient set the http.Client for the [fauna.Client]
func HTTPClient(client *http.Client) ClientConfigFn {
	return func(c *Client) { c.http = client }
}

// Headers specify headers for the [fauna.Client]
func Headers(headers map[string]string) ClientConfigFn {
	return func(c *Client) {
		for k, v := range headers {
			c.headers[k] = v
		}
	}
}

// LastTxnTime toggle if [fauna.Client] records the last transaction time
func LastTxnTime(enabled bool) ClientConfigFn {
	return func(c *Client) {
		c.lastTxnTime.Enabled = enabled
	}
}

// Linearized set header on the [fauna.Client]
// A boolean. If true, unconditionally run the query as strictly serialized/linearized.
// This affects read-only transactions, as transactions which write will be strictly serialized.
func Linearized(enabled bool) ClientConfigFn {
	return func(c *Client) {
		c.SetHeader(HeaderLinearized, fmt.Sprintf("%v", enabled))
	}
}

// MaxContentionRetries set header on the [fauna.Client]
// An integer. The maximum number of times a transaction is retried due to OCC failure.
func MaxContentionRetries(i int) ClientConfigFn {
	return func(c *Client) {
		c.SetHeader(HeaderMaxContentionRetries, fmt.Sprintf("%v", i))
	}
}

// SetHeader update [fauna.Client] header
func (c *Client) SetHeader(key, val string) {
	c.headers[key] = val
}

// QueryTimeout set header on the [fauna.Client]
func QueryTimeout(d time.Duration) ClientConfigFn {
	return func(c *Client) {
		c.SetHeader(HeaderTimeoutMs, fmt.Sprintf("%v", d.Milliseconds()))
	}
}

// Tags sets header on the [fauna.Client]
func Tags(tags map[string]string) ClientConfigFn {
	return func(c *Client) {
		c.SetHeader(HeaderTags, argsStringFromMap(tags))
	}
}

// TypeChecking toggle if [fauna.Client] enforces type checking
func TypeChecking(enabled bool) ClientConfigFn {
	return func(c *Client) {
		c.typeCheckingEnabled = enabled
	}
}

// URL set the [fauna.Client] URL
func URL(url string) ClientConfigFn {
	return func(c *Client) { c.url = url }
}

// QueryOptFn function to set options on the [Client.Query]
type QueryOptFn func(req *fqlRequest)

// QueryContext set the [context.Context] for a single [Client.Query]
func QueryContext(ctx context.Context) QueryOptFn {
	return func(req *fqlRequest) {
		req.Context = ctx
	}
}

// QueryTxnTime toggle if [fauna.Client] records the last transaction for a single [Client.Query]
func QueryTxnTime(enabled bool) QueryOptFn {
	return func(req *fqlRequest) {
		req.TxnTimeEnabled = enabled
	}
}

// QueryTypeChecking toggle if [fauna.Client] uses type checking for a single [Client.Query]
func QueryTypeChecking(enabled bool) QueryOptFn {
	return func(req *fqlRequest) {
		req.Headers[HeaderTypeChecking] = fmt.Sprintf("%v", enabled)
	}
}

// QueryTags set the tags header on a single [Client.Query]
func QueryTags(tags map[string]string) QueryOptFn {
	return func(req *fqlRequest) {
		if val, exists := req.Headers[HeaderTags]; exists {
			req.Headers[HeaderTags] = argsStringFromMap(tags, strings.Split(val, ",")...)
		} else {
			req.Headers[HeaderTags] = argsStringFromMap(tags)
		}
	}
}

// QueryTraceparent sets the header on a single [Client.Query]
func QueryTraceparent(id string) QueryOptFn {
	return func(req *fqlRequest) { req.Headers[HeaderTraceparent] = id }
}

// Timeout set the query timeout on a single [Client.Query]
func Timeout(dur time.Duration) QueryOptFn {
	return func(req *fqlRequest) {
		req.Headers[HeaderTypeChecking] = fmt.Sprintf("%f", dur.Seconds())
	}
}

func argsStringFromMap(input map[string]string, currentArgs ...string) string {
	params := url.Values{}

	for _, c := range currentArgs {
		s := strings.Split(c, "=")
		params.Set(s[0], s[1])
	}

	for k, v := range input {
		params.Set(k, v)
	}

	return strings.ReplaceAll(params.Encode(), "&", ",")
}
