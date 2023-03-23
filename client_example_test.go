package fauna_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fauna/fauna-go"
)

// ExampleDefaultClient query fauna running in a local Docker instance:
//
//	docker run --rm -p 8443:8443 fauna/faunadb:latest
func ExampleDefaultClient() {
	// IMPORTANT: just for the purpose of example, don't actually hardcode secret
	_ = os.Setenv(fauna.EnvFaunaSecret, "secret")
	_ = os.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

	client, clientErr := fauna.NewDefaultClient()
	if clientErr != nil {
		log.Fatalf("client should have been initialized: %s", clientErr.Error())
	}

	query, qErr := fauna.FQL(`Math.abs(12e5)`)
	if qErr != nil {
		log.Fatalf("query failed: %s", qErr.Error())
	}

	var result float32
	_, queryErr := client.Query(query, &result)
	if queryErr != nil {
		log.Fatalf("request failed: %s", queryErr.Error())
	}

	fmt.Printf("%0.f", result)
	// Output: 1200000
}

// ExampleNewClient query fauna running in a local Docker instance:
//
//	docker run --rm -p 8443:8443 fauna/faunadb:latest
func ExampleNewClient() {
	client := fauna.NewClient(
		// IMPORTANT: just for the purpose of example, don't actually hardcode secret
		"secret",
		fauna.HTTPClient(http.DefaultClient),
		fauna.URL(fauna.EndpointLocal),
		fauna.Context(context.Background()),
		fauna.QueryTimeout(time.Minute*3),
	)

	query, qErr := fauna.FQL(`Math.abs(12e5)`, nil)
	if qErr != nil {
		log.Fatalf("query failed: %s", qErr.Error())
	}

	var result float32
	_, queryErr := client.Query(query, &result)
	if queryErr != nil {
		log.Fatalf("request failed: %s", queryErr.Error())
	}

	fmt.Printf("%0.f", result)
	// Output: 1200000
}

// ExampleComposedQuery query fauna running in a local Docker instance:
//
//	docker run --rm -p 8443:8443 fauna/faunadb:latest
func ExampleFQL() {
	// IMPORTANT: just for the purpose of example, don't actually hardcode secret
	_ = os.Setenv(fauna.EnvFaunaSecret, "secret")
	_ = os.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

	client, clientErr := fauna.NewDefaultClient()
	if clientErr != nil {
		log.Fatalf("client should have been initialized: %s", clientErr.Error())
	}

	type MyObj struct {
		fauna.Document
		Name string `fauna:"name"`
	}

	query, fqlErr := fauna.FQL("let x = ${my_obj}\nx { name }", map[string]any{"my_obj": &MyObj{Name: "foo"}})
	if fqlErr != nil {
		log.Fatalf("query failed: %s", fqlErr.Error())
	}

	var result map[string]any
	_, queryErr := client.Query(query, &result)
	if queryErr != nil {
		log.Fatalf("request failed: %s", queryErr.Error())
	}

	fmt.Printf("%s", result["name"])
	// Output: foo
}
