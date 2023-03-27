package fauna

import (
	"context"
	"fmt"
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

// DefaultTypecheck set header on the [fauna.Client]
// Enable or disable typechecking of the query before evaluation. If
// not set, Fauna will use the value of the "typechecked" flag on
// the database configuration.
func DefaultTypecheck(enabled bool) ClientConfigFn {
	return func(c *Client) {
		c.setHeader(HeaderTypecheck, fmt.Sprintf("%v", enabled))
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

// QueryTags sets header on the [fauna.Client]
// Set tags to associate with the query. See [logging]
//
// [logging]: https://docs.fauna.com/fauna/current/build/logs/query_log/
func QueryTags(tags map[string]string) ClientConfigFn {
	return func(c *Client) {
		c.setHeader(HeaderTags, argsStringFromMap(tags))
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

// Tags set the tags header on a single [Client.Query]
func Tags(tags map[string]string) QueryOptFn {
	return func(req *fqlRequest) {
		if val, exists := req.Headers[HeaderTags]; exists {
			req.Headers[HeaderTags] = argsStringFromMap(tags, strings.Split(val, ",")...)
		} else {
			req.Headers[HeaderTags] = argsStringFromMap(tags)
		}
	}
}

// Traceparent sets the header on a single [Client.Query]
func Traceparent(id string) QueryOptFn {
	return func(req *fqlRequest) { req.Headers[HeaderTraceparent] = id }
}

// Timeout set the query timeout on a single [Client.Query]
func Timeout(dur time.Duration) QueryOptFn {
	return func(req *fqlRequest) {
		req.Headers[HeaderTimeoutMs] = fmt.Sprintf("%d", dur.Milliseconds())
	}
}

// Typecheck sets the header on a single [Client.Query]
func Typecheck(enabled bool) QueryOptFn {
	return func(req *fqlRequest) { req.Headers[HeaderTypecheck] = fmt.Sprintf("%v", enabled) }
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
