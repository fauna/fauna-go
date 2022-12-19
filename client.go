package fauna

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

var authorizationHeader = "Authorization"

type Client struct {
	url                 string
	secret              string
	maxConnections      int
	timeoutMilliseconds int
	headers             map[string]string

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

	response.raw = r
	return response
}

func (c *Client) token() string {
	return fmt.Sprintf("Bearer %s", c.secret)
}
