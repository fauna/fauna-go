package fauna_test

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/fauna/fauna-go"
)

// ExampleNewClient query fauna running in a local Docker instance
//
//	docker run --rm -p 8443:8443 fauna/faunadb:fqlx
func ExampleNewClient() {
	client := fauna.NewClient(
		"secret",
		fauna.URL(fauna.EndpointLocal),
		fauna.HTTPClient(http.DefaultClient),
		fauna.Context(context.TODO()),
	)

	var result float32
	_, queryErr := client.Query(`Math.abs(12e5)`, nil, &result)
	if queryErr != nil {
		log.Fatalf(queryErr.Error())
	}

	fmt.Printf("%0.f", result)
	// Output: 1200000
}
