package fauna

import (
	"bytes"
	"encoding/json"
	"errors"
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
	authorizationHeader = "Authorization"
	txnTimeHeader       = "X-Txn-Time"
	lastSeenTxnHeader   = "X-Last-Seen-Txn"
)

type Client struct {
	url                 string
	secret              string
	maxConnections      int
	timeoutMilliseconds int
	headers             map[string]string
	txnTimeEnabled      bool
	lastTxnTime         int64

	tcp      *http.Transport
	http     *http.Client
	endpoint *net.Addr

	// maxRetries?
	// linearized?
	// tags?
	// traceParent?
}

func DefaultClient() (*Client, error) {
	secret := os.Getenv(secretKey)
	if secret == "" {
		err := errors.New(fmt.Sprintf("unable to load key from environment variable '%v'", secretKey))
		return nil, err
	}
	return NewClient(secret, URL(productionUrl), MaxConnections(defaultMaxConnections), TimeoutMilliseconds(defaultTimeoutMilliseconds)), nil
}

func NewClient(secret string, configs ...ClientConfig) *Client {
	client := &Client{secret: secret}

	for _, config := range configs {
		config(client)
	}

	client.tcp = &http.Transport{
		MaxConnsPerHost:       client.maxConnections,
		ResponseHeaderTimeout: time.Duration(client.timeoutMilliseconds) * time.Millisecond,
	}

	client.http = &http.Client{
		Transport: client.tcp,
	}

	return client
}

func (c *Client) Query(fql string, obj any) error {
	req := NewRequest(fql)
	res := c.Do(req)
	if res.err != nil {
		return res.err
	}
	return json.Unmarshal(res.Data, obj)
}

func (c *Client) Do(request *Request) *Response {
	bout, err := json.Marshal(request)
	if err != nil {
		return ErrorResponse(err)
	}

	request.raw, err = http.NewRequest(http.MethodPost, c.url, bytes.NewReader(bout))
	if err != nil {
		return ErrorResponse(err)
	}

	request.raw.Header.Add(authorizationHeader, c.token())
	for k, v := range c.headers {
		request.raw.Header.Add(k, v)
	}

	if c.txnTimeEnabled {
		if lastSeen := atomic.LoadInt64(&c.lastTxnTime); lastSeen != 0 {
			request.raw.Header.Add(lastSeenTxnHeader, strconv.FormatInt(lastSeen, 10))
		}
	}

	r, err := c.http.Do(request.raw)
	if err != nil {
		return ErrorResponse(err)
	}

	bin, err := io.ReadAll(r.Body)
	if err != nil {
		return ErrorResponse(err)
	}

	var response *Response
	err = json.Unmarshal(bin, &response)
	if err != nil {
		return ErrorResponse(err)
	}
	response.raw = r

	err = c.storeLastTxnTime(response.raw.Header)
	if err != nil {
		return ErrorResponse(err)
	}

	response.raw = r
	return response
}

func (c *Client) storeLastTxnTime(header http.Header) (err error) {
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

func parseTxnTimeHeader(header http.Header) (txnTime int64, err error) {
	h := header.Get(txnTimeHeader)
	if h != "" {
		return strconv.ParseInt(h, 10, 64)
	} else {
		return math.MinInt, nil
	}
}

func (c *Client) token() string {
	return fmt.Sprintf("Bearer %s", c.secret)
}
