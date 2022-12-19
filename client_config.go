package fauna

const productionUrl = "https://db.fauna.com/query/1"
const previewUrl = "https://db.fauna-preview.com/query/1"
const localUrl = "http://localhost:8443/query/1"

const secretKey = "FAUNA_KEY"
const defaultMaxConnections = 10
const defaultTimeoutMilliseconds = 60 * 1000

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
