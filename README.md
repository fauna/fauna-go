# Fauna Go

> **Note**
> This driver is in beta release and not recommended for production use. It operates with the Fauna database service via an API which is also in beta release, and is not recommended for production use. This driver is not compatible with v4 or earlier versions of Fauna. If you would like to participate in the private beta program please contact product@fauna.com.

Go driver for Fauna

# Fauna Go Driver

[![Go Report Card](https://goreportcard.com/badge/github.com/fauna/fauna-go)](https://goreportcard.com/report/github.com/fauna/fauna-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/fauna/fauna-go.svg)](https://pkg.go.dev/github.com/fauna/fauna-go)
[![License](https://img.shields.io/badge/license-MPL_2.0-blue.svg?maxAge=2592000)](https://raw.githubusercontent.com/fauna/fauna-go/main/LICENSE)

A Go lang driver for [Fauna](https://fauna.com/).

## Supported Go Versions

Currently, the driver is tested on:
- 1.19
- 1.20

## Using the Driver

### Basic Usage

```go
package main

import (
	"fmt"

	"github.com/fauna/fauna-go/v10/fauna"
)

type Dog struct {
	Name string `fauna:"name"`
}

func main() {
	client := fauna.NewFaunaClient("your-secret-here")

	if _, err := client.Query(`Collection.create({ name: "Dogs" }`); err != nil {
		panic(err)
	}

    var scout Dog
	res, err := client.Query(`Dogs.create({ name: name }`, map[string]any{"name": "Scout"}, &scout)
	if err != nil {
		panic(err)
	}

    if res.Error != nil {
        panic(res.Error)
    }

	fmt.Println(scout)
}
```

### Query Composition

For FQL templates, denote variables with `${}` and pass variables as `map[string]any` to `fauna.FQL()`. You can escape a variable with by prepending
an additional `$`.

```go
package main

import (
	"fmt"

	"github.com/fauna/fauna-go/v10/fauna"
)

func userByTin(tin string) (*fauna.Query, error) {
    return fauna.FQL(`Users.byTin(${tin})`, map[string]any{"tin":tin})
}

func main() {
	client := fauna.NewFaunaClient("your-secret-here")

    byTin, err := userByTin("1234")
    if err != nil {
        panic(err)
    }

    var username string
    res, err := client.Query(`${user}["name"]`, map[string]any{"user": byTin}, &username)
	if err != nil {
		panic(err)
	}

    if res.Error != nil {
        panic(res.Error)
    }

	fmt.Println(username)
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
