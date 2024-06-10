package fauna_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fauna/fauna-go/v2"
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
		fauna.DefaultTimeouts(),
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

func ExampleClient_Stream() {
	// IMPORTANT: just for the purpose of example, don't actually hardcode secret
	_ = os.Setenv(fauna.EnvFaunaSecret, "secret")
	_ = os.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

	client, err := fauna.NewDefaultClient()
	if err != nil {
		log.Fatalf("client should have been initialized: %s", err)
	}

	// setup a collection
	setupQuery, _ := fauna.FQL(`
		if (!Collection.byName('StreamingSandbox').exists()) {
			Collection.create({ name: 'StreamingSandbox' })
        } else {
			StreamingSandbox.all().forEach(.delete())
        }
	`, nil)
	if _, err := client.Query(setupQuery); err != nil {
		log.Fatalf("failed to setup the collection: %s", err)
	}

	// create a stream
	streamQuery, _ := fauna.FQL(`StreamingSandbox.all().toStream()`, nil)
	events, err := client.Stream(streamQuery)
	if err != nil {
		log.Fatalf("failed to subscribe to the stream value: %s", err)
	}
	defer events.Close()

	// produce some events while the subscription is open
	createQuery, _ := fauna.FQL(`StreamingSandbox.create({ foo: 'bar' })`, nil)
	updateQuery, _ := fauna.FQL(`StreamingSandbox.all().forEach(.update({ foo: 'baz' }))`, nil)
	deleteQuery, _ := fauna.FQL(`StreamingSandbox.all().forEach(.delete())`, nil)

	queries := []*fauna.Query{createQuery, updateQuery, deleteQuery}
	for _, query := range queries {
		if _, err := client.Query(query); err != nil {
			log.Fatalf("failed execute CRUD query: %s", err)
		}
	}

	// fetch the produced events
	type Data struct {
		Foo string `fauna:"foo"`
	}

	var event fauna.Event

	expect := 3
	for expect > 0 {
		err := events.Next(&event)
		if err != nil {
			log.Fatalf("failed to receive next event: %s", err)
		}
		switch event.Type {
		case fauna.AddEvent, fauna.UpdateEvent, fauna.RemoveEvent:
			var data Data
			if err := event.Unmarshal(&data); err != nil {
				log.Fatalf("failed to unmarshal event data: %s", err)
			}
			fmt.Printf("Event: %s Data: %+v\n", event.Type, data)
			expect--
		}
	}
	// Output: Event: add Data: {Foo:bar}
	// Event: update Data: {Foo:baz}
	// Event: remove Data: {Foo:baz}
}

func ExampleClient_Subscribe() {
	// IMPORTANT: just for the purpose of example, don't actually hardcode secret
	_ = os.Setenv(fauna.EnvFaunaSecret, "secret")
	_ = os.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

	client, err := fauna.NewDefaultClient()
	if err != nil {
		log.Fatalf("client should have been initialized: %s", err)
	}

	// setup a collection
	setupQuery, _ := fauna.FQL(`
		if (!Collection.byName('StreamingSandbox').exists()) {
			Collection.create({ name: 'StreamingSandbox' })
        } else {
			StreamingSandbox.all().forEach(.delete())
        }
	`, nil)
	if _, err := client.Query(setupQuery); err != nil {
		log.Fatalf("failed to setup the collection: %s", err)
	}

	// create a stream
	streamQuery, _ := fauna.FQL(`StreamingSandbox.all().toStream()`, nil)
	result, err := client.Query(streamQuery)
	if err != nil {
		log.Fatalf("failed to create a stream: %s", err)
	}

	var stream fauna.Stream
	if err := result.Unmarshal(&stream); err != nil {
		log.Fatalf("failed to unmarshal the stream value: %s", err)
	}

	// initiate the stream subscription
	events, err := client.Subscribe(stream)
	if err != nil {
		log.Fatalf("failed to subscribe to the stream value: %s", err)
	}
	defer events.Close()

	// produce some events while the subscription is open
	createQuery, _ := fauna.FQL(`StreamingSandbox.create({ foo: 'bar' })`, nil)
	updateQuery, _ := fauna.FQL(`StreamingSandbox.all().forEach(.update({ foo: 'baz' }))`, nil)
	deleteQuery, _ := fauna.FQL(`StreamingSandbox.all().forEach(.delete())`, nil)

	queries := []*fauna.Query{createQuery, updateQuery, deleteQuery}
	for _, query := range queries {
		if _, err := client.Query(query); err != nil {
			log.Fatalf("failed execute CRUD query: %s", err)
		}
	}

	// fetch the produced events
	type Data struct {
		Foo string `fauna:"foo"`
	}

	var event fauna.Event

	expect := 3
	for expect > 0 {
		err := events.Next(&event)
		if err != nil {
			log.Fatalf("failed to receive next event: %s", err)
		}
		switch event.Type {
		case fauna.AddEvent, fauna.UpdateEvent, fauna.RemoveEvent:
			var data Data
			if err := event.Unmarshal(&data); err != nil {
				log.Fatalf("failed to unmarshal event data: %s", err)
			}
			fmt.Printf("Event: %s Data: %+v\n", event.Type, data)
			expect--
		}
	}
	// Output: Event: add Data: {Foo:bar}
	// Event: update Data: {Foo:baz}
	// Event: remove Data: {Foo:baz}
}
