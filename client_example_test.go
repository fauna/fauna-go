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
		log.Fatalf("client should have been initialized: %s", clientErr)
	}

	query, qErr := fauna.FQL(`Math.abs(12e5)`, nil)
	if qErr != nil {
		log.Fatalf("query failed: %s", qErr)
	}

	res, queryErr := client.Query(query)
	if queryErr != nil {
		log.Fatalf("request failed: %s", queryErr)
	}

	var result float32
	if err := res.Unmarshal(&result); err != nil {
		log.Fatalf("%s", err)
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
		log.Fatalf("query failed: %s", qErr)
	}

	res, queryErr := client.Query(query)
	if queryErr != nil {
		log.Fatalf("request failed: %s", queryErr)
	}

	var result float32
	if err := res.Unmarshal(&result); err != nil {
		log.Fatalf("%s", queryErr)
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
		log.Fatalf("client should have been initialized: %s", clientErr)
	}

	query, fqlErr := fauna.FQL(`2 + 2`, nil)
	if fqlErr != nil {
		log.Fatalf("query failed: %s", fqlErr)
	}

	res, queryErr := client.Query(query)
	if queryErr != nil {
		log.Fatalf("request failed: %s", queryErr)
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
		log.Fatalf("client should have been initialized: %s", clientErr)
	}

	query, fqlErr := fauna.FQL(`${num} + 2`, map[string]any{"num": 2})
	if fqlErr != nil {
		log.Fatalf("query failed: %s", fqlErr)
	}

	res, queryErr := client.Query(query)
	if queryErr != nil {
		log.Fatalf("request failed: %s", queryErr)
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
		log.Fatalf("client should have been initialized: %s", clientErr)
	}

	type myObj struct {
		Value int `fauna:"value"`
	}

	arg := &myObj{Value: 2}

	query, fqlErr := fauna.FQL(`${obj}["value"] + 2`, map[string]any{"obj": arg})
	if fqlErr != nil {
		log.Fatalf("query failed: %s", fqlErr)
	}

	res, queryErr := client.Query(query)
	if queryErr != nil {
		log.Fatalf("request failed: %s", queryErr)
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
		log.Fatalf("client should have been initialized: %s", clientErr)
	}

	type myObj struct {
		Value int `fauna:"value"`
	}

	// Mock out an object that looks like our struct `myObj`.
	query, fqlErr := fauna.FQL(`{"value": 4}`, nil)
	if fqlErr != nil {
		log.Fatalf("query failed: %s", fqlErr)
	}

	res, queryErr := client.Query(query)
	if queryErr != nil {
		log.Fatalf("request failed: %s", queryErr)
	}

	// Unmarshal the resulting object into a `myObj` object.
	var result myObj
	if err := res.Unmarshal(&result); err != nil {
		log.Fatalf("unmarshal failed: %s", queryErr)
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
		log.Fatalf("client should have been initialized: %s", clientErr)
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

func ExampleClient_Paginate() {
	// IMPORTANT: just for the purpose of example, don't actually hardcode secret
	_ = os.Setenv(fauna.EnvFaunaSecret, "secret")
	_ = os.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

	client, clientErr := fauna.NewDefaultClient()
	if clientErr != nil {
		log.Fatalf("client should have been initialized: %s", clientErr)
	}

	collectionName := "pagination_sandbox"

	// create a collection
	deleteQuery, deleteQueryErr := fauna.FQL(`Collection.byName(${coll})?.delete()`, map[string]any{"coll": collectionName})
	if deleteQueryErr != nil {
		log.Fatalf("failed to construct delete query")
	}

	if _, deleteErr := client.Query(deleteQuery); deleteErr != nil {
		log.Fatalf("failed to clean up collection: %t", deleteErr)
	}

	createQuery, createQueryErr := fauna.FQL(`Collection.create({ name: ${name} })`, map[string]any{"name": collectionName})
	if createQueryErr != nil {
		log.Fatalf("failed to construct create query")
	}
	if _, createErr := client.Query(createQuery); createErr != nil {
		log.Fatalf("failed to create collection: %t", createErr)
	}

	// seed collection
	collectionModule := &fauna.Module{Name: collectionName}
	// update Output comment at the bottom if you change this
	totalTestItems := 20

	for i := 0; i < totalTestItems; i++ {
		createCollectionQuery, createItemQueryErr := fauna.FQL(`${mod}.create({ value: ${i} })`, map[string]any{
			"mod": collectionModule,
			"i":   i,
		})
		if createItemQueryErr != nil {
			log.Fatalf("failed to construct create item query: %t", createItemQueryErr)
		}

		if _, createItemErr := client.Query(createCollectionQuery); createItemErr != nil {
			log.Fatalf("failed to create seed item: %t", createItemErr)
		}
	}

	// paginate collection
	paginationQuery, paginationQueryErr := fauna.FQL(`${mod}.all()`, map[string]any{"mod": collectionModule})
	if paginationQueryErr != nil {
		log.Fatalf("failed to construct pagination query: %t", paginationQueryErr)
	}

	type Item struct {
		Value int `fauna:"value"`
	}

	var items []Item

	paginator := client.Paginate(paginationQuery)
	for {
		page, pageErr := paginator.Next()
		if pageErr != nil {
			log.Fatalf("pagination failed: %t", pageErr)
		}

		var pageItems []Item
		if marshalErr := page.Unmarshal(&pageItems); marshalErr != nil {
			log.Fatalf("failed to unmarshal page: %t", marshalErr)
		}

		items = append(items, pageItems...)

		if !paginator.HasNext() {
			break
		}
	}

	fmt.Printf("%d", len(items))
	// Output: 20
}
