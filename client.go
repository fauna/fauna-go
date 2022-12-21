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

var authorizationHeader = "Authorization"
var txnTimeHeader = "X-Txn-Time"
var lastSeenTxnHeader = "X-Last-Seen-Txn"

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

func DefaultClient() *Client {
	secret := os.Getenv(secretKey)
	if secret == "" {
		return nil
	}
	return NewClient(secret, URL(productionUrl), MaxConnections(defaultMaxConnections), TimeoutMilliseconds(defaultTimeoutMilliseconds))
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

func Query[T any](c *Client, request *Request) *Response[T] {
	bout, err := json.Marshal(request)
	if err != nil {
		return ErrorResponse[T](err)
	}

	request.raw, err = http.NewRequest(http.MethodPost, c.url, bytes.NewReader(bout))
	if err != nil {
		return ErrorResponse[T](err)
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
		return ErrorResponse[T](err)
	}

	bin, err := io.ReadAll(r.Body)
	if err != nil {
		return ErrorResponse[T](err)
	}

	var response *Response[T]
	err = json.Unmarshal(bin, &response)
	if err != nil {
		return ErrorResponse[T](err)
	}

	err = c.storeLastTxnTime(response.raw.Header)
	if err != nil {
		return ErrorResponse[T](err)
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
