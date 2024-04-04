# The Official Golang Driver for [Fauna](https://fauna.com/).

[![Go Report Card](https://goreportcard.com/badge/github.com/fauna/fauna-go)](https://goreportcard.com/report/github.com/fauna/fauna-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/fauna/fauna-go.svg)](https://pkg.go.dev/github.com/fauna/fauna-go)
[![License](https://img.shields.io/badge/license-MPL_2.0-blue.svg?maxAge=2592000)](https://raw.githubusercontent.com/fauna/fauna-go/main/LICENSE)

This driver can only be used with FQL v10, and is not compatible with earlier versions of FQL. To query your databases with earlier API versions, see the [faunadb](https://pkg.go.dev/github.com/fauna/faunadb-go/v4) version.

See the [Fauna Documentation](https://docs.fauna.com/fauna/current/) for additional information how to configure and query your databases.

## Supported Go Versions

Currently, the driver is tested on:
- 1.19
- 1.20

## Using the Driver

For FQL templates, denote variables with `${}` and pass variables as `map[string]any` to `fauna.FQL()`. You can escape a variable with by prepending
an additional `$`.

### Basic Usage

```go
package main

import (
	"fmt"

	"github.com/fauna/fauna-go"
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

	"github.com/fauna/fauna-go"
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

	"github.com/fauna/fauna-go"
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

Use the `Client.Paginate()` method to iterate sets that contain more than one page of results.

`Client.paginate()` accepts the same query options as `Client.Query()`.

Change the default items per page using FQL's `<set>.pageSize()` method.

```go
package main

import (
	"fmt"
	"time"

	"github.com/fauna/fauna-go"
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

	options := fauna.Timeout(time.Minute)

	paginator := client.Paginate(query, options)
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

import "github.com/fauna/fauna-go"

func main() {
	client := fauna.NewClient("mysecret", fauna.Timeouts{QueryTimeout: 20 * time.Second})
}
```

#### Client Buffer Timeout

Time beyond `QueryTimeout` at which the client will abort a request if it has not received a response. The default is 5s, which should account for network latency for most clients. The value must be greater than zero. The closer to zero the value is, the more likely the client is to abort the request before the server can report a legitimate response or error.

```go
package main

import "github.com/fauna/fauna-go"

func main() {
	client := fauna.NewClient("mysecret", fauna.Timeouts{ClientBufferTimeout: 20 * time.Second})
}
```

#### Connection Timeout

The amount of time to wait for the connection to complete.

```go
package main

import "github.com/fauna/fauna-go"

func main() {
	client := fauna.NewClient("mysecret", fauna.Timeouts{ConnectionTimeout: 10 * time.Second})
}
```

#### Idle Connection Timeout

The maximum amount of time an idle (keep-alive) connection will remain idle before closing itself.

```go
package main

import "github.com/fauna/fauna-go"

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

import "github.com/fauna/fauna-go"

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

    "github.com/fauna/fauna-go"
)

func main() {
	client := fauna.NewClient("mysecret", fauna.DefaultTimeouts(), fauna.MaxBackoff(10 * time.Second))
}
```

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
