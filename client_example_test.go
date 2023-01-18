package fauna_test

import (
	"fmt"
	"log"
	"os"

	"github.com/fauna/fauna-go"
)

// ExampleNewClient query fauna running in a local Docker instance
//
//	docker run --rm -p 8443:8443 fauna/faunadb:latest
func ExampleNewClient() {
	// IMPORTANT: just for the purpose of example, don't actually hardcode secret
	_ = os.Setenv(fauna.EnvFaunaSecret, "secret")
	_ = os.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

	client, clientErr := fauna.DefaultClient()
	if clientErr != nil {
		log.Fatalf("client should have been initialized: %s", clientErr.Error())
	}

	var result float32
	_, queryErr := client.Query(`Math.abs(12e5)`, nil, &result)
	if queryErr != nil {
		log.Fatalf("query failed: %s", queryErr.Error())
	}

	fmt.Printf("%0.f", result)
	// Output: 1200000
}
