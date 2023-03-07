package fauna

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (c *Client) setHeader(key, val string) {
	c.headers[key] = val
}

// ClientConfigFn configuration options for the [fauna.Client]
type ClientConfigFn func(*Client)

// Context specify the context to be used for the [fauna.Client]
func Context(ctx context.Context) ClientConfigFn {
	return func(c *Client) { c.ctx = ctx }
}

// HTTPClient set the http.Client for the [fauna.Client]
func HTTPClient(client *http.Client) ClientConfigFn {
	return func(c *Client) { c.http = client }
}

// AdditionalHeaders specify headers for the [fauna.Client]
func AdditionalHeaders(headers map[string]string) ClientConfigFn {
	return func(c *Client) {
		for k, v := range headers {
			c.headers[k] = v
		}
	}
}

// Linearized set header on the [fauna.Client]
// If true, unconditionally run the query as strictly serialized.
// This affects read-only transactions. Transactions which write will always be strictly serialized.
func Linearized(enabled bool) ClientConfigFn {
	return func(c *Client) {
		c.setHeader(HeaderLinearized, fmt.Sprintf("%v", enabled))
	}
}

// MaxContentionRetries set header on the [fauna.Client]
// The max number of times to retry the query if contention is encountered.
func MaxContentionRetries(i int) ClientConfigFn {
	return func(c *Client) {
		c.setHeader(HeaderMaxContentionRetries, fmt.Sprintf("%v", i))
	}
}

// QueryTimeout set header on the [fauna.Client]
func QueryTimeout(d time.Duration) ClientConfigFn {
	return func(c *Client) {
		c.setHeader(HeaderTimeoutMs, fmt.Sprintf("%v", d.Milliseconds()))
	}
}

// Tags sets header on the [fauna.Client]
// Set tags to associate with the query. See [logging]
//
// [logging]: https://docs.fauna.com/fauna/current/build/logs/query_log/
func Tags(tags map[string]string) ClientConfigFn {
	return func(c *Client) {
		c.setHeader(HeaderTags, argsStringFromMap(tags))
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
