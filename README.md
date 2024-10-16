# Official Golang Driver for [Fauna v10](https://fauna.com/) (current)

[![Go Report Card](https://goreportcard.com/badge/github.com/fauna/fauna-go)](https://goreportcard.com/report/github.com/fauna/fauna-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/fauna/fauna-go.svg)](https://pkg.go.dev/github.com/fauna/fauna-go/v3)
[![License](https://img.shields.io/badge/license-MPL_2.0-blue.svg?maxAge=2592000)](https://raw.githubusercontent.com/fauna/fauna-go/main/LICENSE)

This driver can only be used with FQL v10, and is not compatible with earlier versions of FQL. To query your databases with earlier API versions, see the [faunadb](https://pkg.go.dev/github.com/fauna/faunadb-go/v4) version.

See the [Fauna Documentation](https://docs.fauna.com/fauna/current/) for additional information how to configure and query your databases.

## Supported Go Versions

Currently, the driver is tested on:
- 1.19
- 1.20
- 1.21
- 1.22
- 1.23

## API reference

API reference documentation for the driver is available on [pkg.go.dev](https://pkg.go.dev/github.com/fauna/fauna-go/v3#section-documentation).

## Using the Driver

For FQL templates, denote variables with `${}` and pass variables as `map[string]any` to `fauna.FQL()`. You can escape a variable with by prepending
an additional `$`.

### Basic Usage

```go
package main

import (
	"fmt"

	"github.com/fauna/fauna-go/v3"
)

func main() {
	client, clientErr := fauna.NewDefaultClient()
	if clientErr != nil {
		panic(clientErr)
	}

	createColl, _ := fauna.FQL(`Collection.create({ name: "Dogs" })`, nil)
	if _, err := client.Query(createColl); err != nil {
		panic(err)
	}

	createDog, _ := fauna.FQL(`Dogs.create({ name: ${name}})`, map[string]any{"name": "Scout"})
	res, err := client.Query(createDog)
	if err != nil {
		panic(err)
	}

	fmt.Println(res.Data.(*fauna.Document).Data["name"])
}
```

### Using Structs

```go
package main

import (
	"fmt"

	"github.com/fauna/fauna-go/v3"
)

type Dog struct {
	Name string `fauna:"name"`
}

func main() {
	client, clientErr := fauna.NewDefaultClient()
	if clientErr != nil {
		panic(clientErr)
	}

	createColl, _ := fauna.FQL(`Collection.create({ name: "Dogs" })`, nil)
	if _, err := client.Query(createColl); err != nil {
		panic(err)
	}

	newDog := Dog{"Scout"}
	createDog, _ := fauna.FQL(`Dogs.create(${dog})`, map[string]any{"dog": newDog})
	res, err := client.Query(createDog)
	if err != nil {
		panic(err)
	}

	var scout Dog
	if err := res.Unmarshal(&scout); err != nil {
		panic(err)
	}

	fmt.Println(scout.Name)
}
```

### Composing Multiple Queries

```go
package main

import (
	"fmt"

	"github.com/fauna/fauna-go/v3"
)

func addTwo(x int) *fauna.Query {
	q, _ := fauna.FQL(`${x} + 2`, map[string]any{"x": x})
	return q
}

func main() {
	client, clientErr := fauna.NewDefaultClient()
	if clientErr != nil {
		panic(clientErr)
	}

	q, _ := fauna.FQL(`${y} + 4`, map[string]any{"y": addTwo(2)})
	res, err := client.Query(q)
	if err != nil {
		panic(err)
	}

	data := res.Data.(int64)
	fmt.Println(data) // 8
}
```

## Pagination

Use the `Paginate()` method to iterate sets that contain more than one page of results.

`Paginate()` accepts the same query options as `Query()`.

Change the default items per page using FQL's `pageSize()` method.

```go
package main

import (
	"fmt"

	"github.com/fauna/fauna-go/v3"
)

type Product struct {
	Description string `fauna:"description"`
}

func main() {
	client, clientErr := fauna.NewDefaultClient()
	if clientErr != nil {
		panic(clientErr)
	}

	// Adjust `pageSize()` size as needed.
	query, _ := fauna.FQL(`
		Product
			.byName("limes")
			.pageSize(2) { description }`, nil)

	paginator := client.Paginate(query)
	for {
		page, _ := paginator.Next()

		var pageItems []Product
		page.Unmarshal(&pageItems)

		for _, item := range pageItems {
			fmt.Println(item)
		}

		if !paginator.HasNext() {
			break
		}
	}
}
```

## Client Configuration

### Timeouts

#### Query Timeout

The timeout of each query. This controls the maximum amount of time Fauna will execute your query before marking it failed.

```go
package main

import "github.com/fauna/fauna-go/v3"

func main() {
	client := fauna.NewClient("mysecret", fauna.Timeouts{QueryTimeout: 20 * time.Second})
}
```

#### Client Buffer Timeout

Time beyond `QueryTimeout` at which the client will abort a request if it has not received a response. The default is 5s, which should account for network latency for most clients. The value must be greater than zero. The closer to zero the value is, the more likely the client is to abort the request before the server can report a legitimate response or error.

```go
package main

import "github.com/fauna/fauna-go/v3"

func main() {
	client := fauna.NewClient("mysecret", fauna.Timeouts{ClientBufferTimeout: 20 * time.Second})
}
```

#### Connection Timeout

The amount of time to wait for the connection to complete.

```go
package main

import "github.com/fauna/fauna-go/v3"

func main() {
	client := fauna.NewClient("mysecret", fauna.Timeouts{ConnectionTimeout: 10 * time.Second})
}
```

#### Idle Connection Timeout

The maximum amount of time an idle (keep-alive) connection will remain idle before closing itself.

```go
package main

import "github.com/fauna/fauna-go/v3"

func main() {
	client := fauna.NewClient("mysecret", fauna.Timeouts{IdleConnectionTimeout: 10 * time.Second})
}
```

### Retries

By default the client will automatically retry a query if the request results in an HTTP status code 429. Retries use an exponential backoff. The maximum number of retries and maximum wait time before a retry can be configured on the client.

#### Maximum Attempts

The maximum number of times the client will try a query. The default is 3.

```go
package main

import "github.com/fauna/fauna-go/v3"

func main() {
	client := fauna.NewClient("mysecret", fauna.DefaultTimeouts(), fauna.MaxAttempts(1))
}
```

#### Maximum Backoff Time

The maximum amount of time to wait before retrying a query. Retries will use an exponential backoff up to this value. The default is 20 seconds.

```go
package main

import (
	"time"

	"github.com/fauna/fauna-go/v3"
)

func main() {
	client := fauna.NewClient("mysecret", fauna.DefaultTimeouts(), fauna.MaxBackoff(10 * time.Second))
}
```


## Event Streaming

The driver supports [Event
Streaming](https://docs.fauna.com/fauna/current/learn/cdc/#event-streaming).


### Start a stream

An Event Stream lets you consume events from an [event
source](https://docs.fauna.com/fauna/current/learn/cdc/#create-an-event-source)
as a real-time subscription.

To get an event, append
[`eventSource()`](https://docs.fauna.com/fauna/current/reference/reference/schema_entities/set/eventsource)
or
[`eventsOn()`](https://docs.fauna.com/fauna/current/reference/reference/schema_entities/set/eventson)
to a [supported Set](https://docs.fauna.com/fauna/current/reference/cdc/#sets).

To start and subscribe to the stream, pass a query that produces an event source
to `Stream()`:

```go
type Product struct {
	Name			string	`fauna:"name"`
	Description		string	`fauna:"description"`
	Price			float64	`fauna:"price"`
}

func main() {
	client, clientErr := fauna.NewDefaultClient()
	if clientErr != nil {
		panic(clientErr)
	}

	streamQuery, _ := fauna.FQL("Product.all().eventSource()", nil)
	events, err := client.Stream(streamQuery)
	if err != nil {
		panic(err)
	}
	defer events.Close()

	var event fauna.Event
	for {
		err := events.Next(&event)
		if err != nil {
			panic(err)
		}

		switch event.Type {
		case fauna.AddEvent, fauna.UpdateEvent, fauna.RemoveEvent:
			var product Product
			if err = event.Unmarshal(&product); err != nil {
				panic(err)
			}
			fmt.Println(product)
		}
	}
}
```

In query results, the driver represents event sources as `fauna.Stream`
values.

To start a stream from a query result, call `Subscribe()` on a
`fauna.Stream` value. This lets you output a stream alongside normal query
results:

```go
type Product struct {
	Name			string	`fauna:"name"`
	Description		string	`fauna:"description"`
	Price			float64	`fauna:"price"`
}

func main() {
	client, clientErr := fauna.NewDefaultClient()
	if clientErr != nil {
		panic(clientErr)
	}

	dataLoad, _ := fauna.FQL(`
		let products = Product.all()
		{
			Products: products.toArray(),
			Stream: products.eventSource()
		}
	`, nil)

	data, err := client.Query(dataLoad)
	if err != nil {
		panic(err)
	}

	queryResult := struct {
		Products []Product
		Stream   fauna.Stream
	}{}

	if err := data.Unmarshal(&queryResult); err != nil {
		panic(err)
	}

	fmt.Println("Existing products:")
	for _, product := range queryResult.Products {
		fmt.Println(product)
	}

	events, err := client.Subscribe(queryResult.Stream)
	if err != nil {
		panic(err)
	}
	defer events.Close()

	fmt.Println("Products from streaming:")
	var event fauna.Event
	for {
		err := events.Next(&event)
		if err != nil {
			panic(err)
		}
		switch event.Type {
		case fauna.AddEvent, fauna.UpdateEvent, fauna.RemoveEvent:
			var product Product
			if err = event.Unmarshal(&product); err != nil {
				panic(err)
			}
			fmt.Println(product)
		}
	}
}
```


### Stream options

The [client configuration](#client-configuration) sets default query options for
`Stream()`. To override these options, see [query options](#query-options).

The `Subscribe()` method accepts the `fauna.StartTime` and `fauna.EventCursor`
function. Use `fauna.StartTime` to restart a stream at a specific timestamp.

```go
streamQuery, _ := fauna.FQL(`Product.all().eventSource()`, nil)
client.Subscribe(streamQuery, fauna.StartTime(1710968002310000))
```

Use `fauna.EventCursor` to resume a stream after a disconnect:

```go
streamQuery, _ := fauna.FQL(`Product.all().eventSource()`, nil)
client.Subscribe(streamQuery, fauna.EventCursor("abc2345=="))
```

| Function | Description |
| -------- | ----------- |
| `fauna.StartTime`  | Sets the stream start time. Accepts an `int64` representing the start time in microseconds since the Unix epoch.<br><br>The start time must be later than the creation time of the event source. The period between the stream restart and the start time argument can't exceed the `history_days` value for source set's collection. If a collection's `history_days` is `0` or unset, the period can't exceed 15 minutes. |
| `fauna.EventCursor`  | Resumes the stream after the given event cursor. Accepts a `string` representation of the cursor retrieved from a `fauna.Event`. |

## Debug Mode

To enable debug logging, set the `FAUNA_DEBUG` environment variable to an integer for the value of the desired [slog level](https://pkg.go.dev/log/slog#Level).

- `slog.LevelInfo` will log all HTTP responses from Fauna.
- `slog.LevelDebug` will not redact the Authorization header and include the request body.

For Go versions before 1.21 we use a [log.Logger](https://pkg.go.dev/log#Logger) and 1.21+ we use the [slog.Logger](https://pkg.go.dev/log/slog#Logger) -- you can optionally define your own Logger. See `CustomerLoggrer` in [logging_slog_test.go](logging_slog_test.go) for an example.


## Contributing

GitHub pull requests are very welcome.

## LICENSE

Copyright 2023 [Fauna, Inc.](https://fauna.com/)

Licensed under the Mozilla Public License, Version 2.0 (the
"License"); you may not use this software except in compliance with
the License. You may obtain a copy of the License at

[http://mozilla.org/MPL/2.0/](http://mozilla.org/MPL/2.0/)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
implied. See the License for the specific language governing
permissions and limitations under the License.
