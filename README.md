> **Warning**
> This driver is in beta release and not recommended for production use. It operates with the Fauna database service via an API which is also in beta release, and is not recommended for production use. This driver is not compatible with v4 or earlier versions of Fauna. Please feel free to contact product@fauna.com to learn about our special Early Access program for FQL v10.

# A Golang driver for [Fauna](https://fauna.com/).

[![Go Report Card](https://goreportcard.com/badge/github.com/fauna/fauna-go)](https://goreportcard.com/report/github.com/fauna/fauna-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/fauna/fauna-go.svg)](https://pkg.go.dev/github.com/fauna/fauna-go)
[![License](https://img.shields.io/badge/license-MPL_2.0-blue.svg?maxAge=2592000)](https://raw.githubusercontent.com/fauna/fauna-go/main/LICENSE)

See the [Fauna Documentation](https://fqlx-beta--fauna-docs.netlify.app/fqlx/beta/) for additional information how to configure and query your databases.

This driver can only be used with FQL v10, and is not compatible with earlier versions of FQL. To query your databases with earlier API versions, see the [faunadb](https://github.com/fauna/faunadb-go) version.

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

## Client Configuration

### Timeouts

#### Query Timeout

The timeout of each query. This controls the maximum amount of time Fauna will execute your query before marking it failed.

```go
package main

import "github.com/fauna/fauna-go"

func main() {
	client := NewClient("mysecret", Timeouts{QueryTimeout: 20 * time.Second})
}
```

#### Client Buffer Timeout

Time beyond `QueryTimeout` at which the client will abort a request if it has not received a response. The default is 5s, which should account for network latency for most clients. The value must be greater than zero. The closer to zero the value is, the more likely the client is to abort the request before the server can report a legitimate response or error.

```go
package main

import "github.com/fauna/fauna-go"

func main() {
	client := NewClient("mysecret", Timeouts{ClientBufferTimeout: 20 * time.Second})
}
```

#### Connection Timeout

The amount of time to wait for the connection to complete.

```go
package main

import "github.com/fauna/fauna-go"

func main() {
	client := NewClient("mysecret", Timeouts{ConnectionTimeout: 10 * time.Second})
}
```

#### Idle Connection Timeout

The maximum amount of time an idle (keep-alive) connection will remain idle before closing itself.

```go
package main

import "github.com/fauna/fauna-go"

func main() {
	client := NewClient("mysecret", Timeouts{IdleConnectionTimeout: 10 * time.Second})
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
