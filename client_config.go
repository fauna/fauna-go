package fauna

const (
	productionUrl = "https://db.fauna.com/query/1"
	previewUrl    = "https://db.fauna-preview.com/query/1"
	localUrl      = "http://localhost:8443/query/1"

	secretKey                  = "FAUNA_KEY"
	defaultMaxConnections      = 10
	defaultTimeoutMilliseconds = 60 * 10004
)

type ClientConfig func(*Client)

func URL(url string) ClientConfig {
	return func(c *Client) { c.url = url }
}

func MaxConnections(maxConnections int) ClientConfig {
	return func(c *Client) { c.maxConnections = maxConnections }
}

func TimeoutMilliseconds(timeoutMilliseconds int) ClientConfig {
	return func(c *Client) { c.timeoutMilliseconds = timeoutMilliseconds }
}

func Headers(headers map[string]string) ClientConfig {
	return func(c *Client) { c.headers = headers }
}
