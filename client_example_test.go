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

// ExampleNewDefaultClient query fauna running in a local Docker instance:
//
//	docker run --rm -p 8443:8443 fauna/faunadb:latest
func ExampleNewDefaultClient() {
	// IMPORTANT: just for the purpose of example, don't actually hardcode secret
	_ = os.Setenv(fauna.EnvFaunaSecret, "secret")
	_ = os.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

	client, clientErr := fauna.NewDefaultClient()
	if clientErr != nil {
		log.Fatalf("client should have been initialized: %s", clientErr.Error())
	}

	query, qErr := fauna.FQL(`Math.abs(12e5)`, nil)
	if qErr != nil {
		log.Fatalf("query failed: %s", qErr.Error())
	}

	res, queryErr := client.Query(query)
	if queryErr != nil {
		log.Fatalf("request failed: %s", queryErr.Error())
	}

	var result float32
	if err := res.Unmarshal(&result); err != nil {
		log.Fatalf("%s", err.Error())
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

	res, queryErr := client.Query(query)
	if queryErr != nil {
		log.Fatalf("request failed: %s", queryErr.Error())
	}

	var result float32
	if err := res.Unmarshal(&result); err != nil {
		log.Fatalf("%s", queryErr.Error())
	}

	fmt.Printf("%0.f", result)
	// Output: 1200000
}

// ExampleFQL query fauna running in a local Docker instance:
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

	query, fqlErr := fauna.FQL(`2 + 2`, nil)
	if fqlErr != nil {
		log.Fatalf("query failed: %s", fqlErr.Error())
	}

	res, queryErr := client.Query(query)
	if queryErr != nil {
		log.Fatalf("request failed: %s", queryErr.Error())
	}

	fmt.Printf("%d", res.Data)
	// Output: 4
}

func ExampleFQL_arguments() {
	// IMPORTANT: just for the purpose of example, don't actually hardcode secret
	_ = os.Setenv(fauna.EnvFaunaSecret, "secret")
	_ = os.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

	client, clientErr := fauna.NewDefaultClient()
	if clientErr != nil {
		log.Fatalf("client should have been initialized: %s", clientErr.Error())
	}

	query, fqlErr := fauna.FQL(`${num} + 2`, map[string]any{"num": 2})
	if fqlErr != nil {
		log.Fatalf("query failed: %s", fqlErr.Error())
	}

	res, queryErr := client.Query(query)
	if queryErr != nil {
		log.Fatalf("request failed: %s", queryErr.Error())
	}

	fmt.Printf("%d", res.Data)
	// Output: 4
}

func ExampleFQL_structs() {
	// IMPORTANT: just for the purpose of example, don't actually hardcode secret
	_ = os.Setenv(fauna.EnvFaunaSecret, "secret")
	_ = os.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

	client, clientErr := fauna.NewDefaultClient()
	if clientErr != nil {
		log.Fatalf("client should have been initialized: %s", clientErr.Error())
	}

	type myObj struct {
		Value int `fauna:"value"`
	}

	arg := &myObj{Value: 2}

	query, fqlErr := fauna.FQL(`${obj}["value"] + 2`, map[string]any{"obj": arg})
	if fqlErr != nil {
		log.Fatalf("query failed: %s", fqlErr.Error())
	}

	res, queryErr := client.Query(query)
	if queryErr != nil {
		log.Fatalf("request failed: %s", queryErr.Error())
	}

	fmt.Printf("%d", res.Data)
	// Output: 4
}

func ExampleFQL_unmarshal() {
	// IMPORTANT: just for the purpose of example, don't actually hardcode secret
	_ = os.Setenv(fauna.EnvFaunaSecret, "secret")
	_ = os.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

	client, clientErr := fauna.NewDefaultClient()
	if clientErr != nil {
		log.Fatalf("client should have been initialized: %s", clientErr.Error())
	}

	type myObj struct {
		Value int `fauna:"value"`
	}

	// Mock out an object that looks like our struct `myObj`.
	query, fqlErr := fauna.FQL(`{"value": 4}`, nil)
	if fqlErr != nil {
		log.Fatalf("query failed: %s", fqlErr.Error())
	}

	res, queryErr := client.Query(query)
	if queryErr != nil {
		log.Fatalf("request failed: %s", queryErr.Error())
	}

	// Unmarshal the resulting object into a `myObj` object.
	var result myObj
	if err := res.Unmarshal(&result); err != nil {
		log.Fatalf("unmarshal failed: %s", queryErr.Error())
	}

	fmt.Printf("%+v", result)
	// Output: {Value:4}
}

func ExampleFQL_composed() {
	// IMPORTANT: just for the purpose of example, don't actually hardcode secret
	_ = os.Setenv(fauna.EnvFaunaSecret, "secret")
	_ = os.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

	client, clientErr := fauna.NewDefaultClient()
	if clientErr != nil {
		log.Fatalf("client should have been initialized: %s", clientErr.Error())
	}

	type myObj struct {
		Value int `fauna:"value"`
	}

	arg := &myObj{Value: 4}

	// Build a query to pull a value from some object. This could be document already
	// in Fauna.
	getValQuery, gvqErr := fauna.FQL(`${obj}["value"]`, map[string]any{"obj": arg})
	if gvqErr != nil {
		log.Fatalf("query failed: %s", gvqErr)
	}

	// Compose the value query with a multiplier to multiply the value we pulled by
	// some number.
	query, fqlErr := fauna.FQL("${multiplier} * ${value}", map[string]any{
		"value":      getValQuery,
		"multiplier": 4,
	})
	if fqlErr != nil {
		log.Fatalf("query failed: %s", fqlErr)
	}

	res, queryErr := client.Query(query, fauna.Typecheck(true))
	if queryErr != nil {
		log.Fatalf("request failed: %s", queryErr)
	}

	fmt.Printf("%+v", res.Data)
	// Output: 16
}
