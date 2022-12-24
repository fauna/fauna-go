package fauna

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

const (
	EndpointProduction = "https://db.fauna.com/query/1"
	EndpointPreview    = "https://db.fauna-preview.com/query/1"

	EnvFaunaEndpoint         = "FAUNA_ENDPOINT"
	EnvFaunaKey              = "FAUNA_KEY"
	EnvFaunaTypeCheckEnabled = "FAUNA_TYPE_CHECK_ENABLED"

	DefaultMaxConnections = 10
	DefaultTimeout        = 60 * time.Second

	HeaderAuthorization = "Authorization"
	HeaderTxnTime       = "X-Txn-Time"
	HeaderLastSeenTxn   = "X-Last-Seen-Txn"
)

type ClientConfigFn func(*Client)

func URL(url string) ClientConfigFn {
	return func(c *Client) { c.url = url }
}

func HTTPClient(client *http.Client) ClientConfigFn {
	return func(c *Client) { c.http = client }
}

func Headers(headers map[string]string) ClientConfigFn {
	return func(c *Client) { c.headers = headers }
}

type Client struct {
	url            string
	secret         string
	headers        map[string]string
	txnTimeEnabled bool
	lastTxnTime    int64
	forceTypeCheck bool

	tcp      *http.Transport
	http     *http.Client
	endpoint *net.Addr

	// maxRetries?
	// linearized?
	// tags?
	// traceParent?
}

func DefaultClient() (*Client, error) {
	secret, found := os.LookupEnv(EnvFaunaKey)
	if !found {
		return nil, fmt.Errorf("unable to load key from environment variable '%s'", EnvFaunaKey)
	}

	url, urlFound := os.LookupEnv(EnvFaunaEndpoint)
	if !urlFound {
		url = EndpointProduction
	}

	return NewClient(
		secret,
		URL(url),
		HTTPClient(&http.Client{
			Transport: &http.Transport{
				MaxConnsPerHost:       DefaultMaxConnections,
				ResponseHeaderTimeout: DefaultTimeout,
			},
		}),
		Headers(map[string]string{
			HeaderAuthorization: fmt.Sprintf("Bearer %s", secret),
		}),
	), nil
}

func NewClient(secret string, configFns ...ClientConfigFn) *Client {
	client := &Client{secret: secret}
	for _, configFn := range configFns {
		configFn(client)
	}

	return client
}

// Query invoke `fql`
func (c *Client) Query(fql string, args map[string]interface{}, obj any) error {
	res, err := c.do(NewRequest(fql, args))
	if err != nil {
		return err
	}

	if unmarshalErr := json.Unmarshal(res.Data, &obj); unmarshalErr != nil {
		return unmarshalErr
	}

	if res.Error != nil {
		return fmt.Errorf("%s", res.Error.Message)
	}

	return nil
}

func (c *Client) do(request *Request) (*Response, error) {
	bytesOut, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	request.Raw, err = http.NewRequest(http.MethodPost, c.url, bytes.NewReader(bytesOut))
	if err != nil {
		return nil, err
	}

	request.Raw.Header.Add(HeaderAuthorization, fmt.Sprintf("Bearer %s", c.secret))
	for k, v := range c.headers {
		request.Raw.Header.Add(k, v)
	}

	if c.txnTimeEnabled {
		if lastSeen := atomic.LoadInt64(&c.lastTxnTime); lastSeen != 0 {
			request.Raw.Header.Add(HeaderLastSeenTxn, strconv.FormatInt(lastSeen, 10))
		}
	}

	if c.forceTypeCheck {
		request.TypeCheck = true
	}

	r, reqErr := c.http.Do(request.Raw)
	if reqErr != nil {
		return nil, reqErr
	}
	defer func() {
		_ = request.Raw.Body.Close()
	}()

	bin, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		return nil, readErr
	}

	var response Response
	if unmarshalErr := json.Unmarshal(bin, &response); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	response.Raw = r

	if txnTimeErr := c.storeLastTxnTime(response.Raw.Header); txnTimeErr != nil {
		return nil, txnTimeErr
	}

	return &response, nil
}

func (c *Client) storeLastTxnTime(header http.Header) error {
	if c.txnTimeEnabled {
		t, err := parseTxnTimeHeader(header)
		if err != nil {
			return err
		}
		c.syncLastTxnTime(t)
	}

	return nil
}

func (c *Client) syncLastTxnTime(newTxnTime int64) {
	if c.txnTimeEnabled {
		for {
			oldTxnTime := atomic.LoadInt64(&c.lastTxnTime)
			if oldTxnTime >= newTxnTime ||
				atomic.CompareAndSwapInt64(&c.lastTxnTime, oldTxnTime, newTxnTime) {
				break
			}
		}
	}
}

func parseTxnTimeHeader(header http.Header) (int64, error) {
	if h := header.Get(HeaderTxnTime); h != "" {
		return strconv.ParseInt(h, 10, 64)
	}

	return math.MinInt, nil
}
