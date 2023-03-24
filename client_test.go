package fauna_test

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/fauna/fauna-go"
	"github.com/stretchr/testify/assert"
)

func TestDefaultClient(t *testing.T) {
	t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)
	t.Setenv(fauna.EnvFaunaSecret, "secret")

	client, clientErr := fauna.NewDefaultClient()
	if clientErr != nil {
		t.Fatalf("should be able to init default client: %s", clientErr.Error())
	}

	t.Run("should have version", func(t *testing.T) {
		if fauna.DriverVersion == "" {
			t.Errorf("driver version should not be empty")
		}
	})

	t.Run("basic requests", func(t *testing.T) {
		t.Run("String Length Request", func(t *testing.T) {
			s := "foo"

			query, _ := fauna.FQL(`${arg0}.length`, map[string]any{"arg0": s})

			res, queryErr := client.Query(query)
			if queryErr != nil {
				t.Errorf("%s", queryErr.Error())
			}

			var i int
			if err := res.Unmarshal(&i); err != nil {
				t.Errorf(err.Error())
			}

			n := len(s)
			if n != i {
				t.Errorf("expected [%d] got [%d]", n, i)
			}

			t.Run("response has expected stats headers", func(t *testing.T) {
				if res.Stats.ComputeOps == 0 {
					t.Errorf("expected some compute ops")
				}

				// This can be flakey on fast systems
				// if res.Stats.QueryTimeMs == 0 {
				// 	t.Errorf("should have some query time")
				// }

				if res.Stats.ContentionRetries > 0 {
					t.Errorf("should not have any retries")
				}

				if res.Stats.ReadOps > 0 || res.Stats.WriteOps > 0 {
					t.Errorf("should not have read/written any bytes")
				}

				if res.Stats.StorageBytesRead > 0 || res.Stats.StorageBytesWrite > 0 {
					t.Errorf("should not have accessed storage")
				}
			})
		})

		t.Run("Query with options", func(t *testing.T) {
			q, _ := fauna.FQL(`Math.abs(-5.123e3)`)
			res, queryErr := client.Query(q, fauna.Timeout(time.Second))
			if queryErr != nil {
				t.Errorf("query failed: %s", queryErr.Error())
			}

			if res != nil {
				t.Logf("summary: %s", res.Summary)
			}
		})
	})
}

func TestNewClient(t *testing.T) {
	t.Run("default client", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "secret")
		_, clientErr := fauna.NewDefaultClient()
		if clientErr != nil {
			t.Errorf("should be able to init default client: %s", clientErr.Error())
		}
	})

	t.Run("stringify", func(t *testing.T) {
		client := fauna.NewClient("secret", fauna.URL(fauna.EndpointLocal))
		if client.String() != fauna.EndpointLocal {
			t.Errorf("client toString should be equal to the endpoint to ensure we don't expose secrets")
		}
	})

	t.Run("missing secret", func(t *testing.T) {
		_, clientErr := fauna.NewDefaultClient()
		if clientErr == nil {
			t.Errorf("should have failed due to missing secret")
		}
	})

	t.Run("has transaction time", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "secret")
		t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

		client, clientErr := fauna.NewDefaultClient()
		if clientErr != nil {
			t.Fatalf("should be able to init a client: %s", clientErr.Error())
		}

		before := client.GetLastTxnTime()
		if before != 0 {
			t.Fatalf("shouldn't have a transaction time")
		}

		q, _ := fauna.FQL(`Math.abs(-5.123e3)`)
		_, queryErr := client.Query(q)
		if queryErr != nil {
			t.Fatalf("query shouldn't error: %s", queryErr.Error())
		}

		first := client.GetLastTxnTime()
		if first == 0 {
			t.Errorf("should have a last transaction time greater than 0, got: %d", first)
		}

		second := client.GetLastTxnTime()
		if first != second {
			t.Errorf("transaction time not have changed, first [%d] second [%d]", before, second)
		}
	})

	t.Run("custom HTTP client", func(t *testing.T) {
		client := fauna.NewClient(
			"secret",
			fauna.URL(fauna.EndpointLocal),
			fauna.HTTPClient(http.DefaultClient),
		)
		q, _ := fauna.FQL(`Math.abs(-5.123e3)`)
		_, queryErr := client.Query(q)
		if queryErr != nil {
			t.Errorf("failed to query: %s", queryErr.Error())
		}
	})
}

func TestBasicCRUDRequests(t *testing.T) {
	t.Setenv(fauna.EnvFaunaSecret, "secret")
	t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)
	client, err := fauna.NewDefaultClient()
	if err != nil {
		t.Fatalf("%s", err.Error())
	}

	coll := fmt.Sprintf("Person_%v", randomString(12))
	collMod := &fauna.Module{coll}

	t.Run("Create a collection", func(t *testing.T) {
		q, _ := fauna.FQL(`Collection.create({ name: ${name} })`, map[string]any{"name": coll})
		_, queryErr := client.Query(q)
		assert.NoError(t, queryErr)
	})

	n := "John Smith"
	p := &Person{
		Name:    n,
		Address: "123 Range Road Houston, TX 77056",
	}

	t.Run("Create a Person", func(t *testing.T) {
		q, _ := fauna.FQL(`${coll}.create(${person})`, map[string]any{"coll": collMod, "person": p})
		_, queryErr := client.Query(q)
		assert.NoError(t, queryErr)
	})

	t.Run("Query a Person", func(t *testing.T) {
		q, _ := fauna.FQL(`${coll}.all().firstWhere(.name == ${name})`, map[string]any{"coll": collMod, "name": n})
		res, queryErr := client.Query(q)
		if !assert.NoError(t, queryErr) {
			return
		}

		var result Person
		if err := res.Unmarshal(&result); err != nil {
			t.Errorf(err.Error())
		}

		if p.Name != result.Name {
			t.Errorf("expected Name [%s] got [%s]", p.Name, result.Name)
		}
	})

	t.Run("Update a Person", func(t *testing.T) {
		q, _ := fauna.FQL(
			`${coll}.all().firstWhere(.name == ${name}).update({address: "321 Rainy St Seattle, WA 98011"})`,
			map[string]any{"coll": collMod, "name": n})

		res, queryErr := client.Query(q)
		if !assert.NoError(t, queryErr) {
			return
		}

		var result Person
		if err := res.Unmarshal(&result); err != nil {
			t.Errorf(err.Error())
		}

		if p.Address == result.Address {
			t.Errorf("expected [%s] got [%s]", p.Address, result.Address)
		}
	})

	t.Run("Delete a Person", func(t *testing.T) {
		q, _ := fauna.FQL(`${coll}.all().firstWhere(.name == ${name}).delete()`, map[string]any{"coll": collMod, "name": p.Name})
		res, queryErr := client.Query(q)
		if !assert.NoError(t, queryErr) {
			return
		}

		var result Person
		if err := res.Unmarshal(&result); err != nil {
			t.Errorf(err.Error())
		}
	})

	t.Run("Delete a Collection", func(t *testing.T) {
		q, _ := fauna.FQL(`Collection.byName(${coll}).delete()`, map[string]any{"coll": coll})
		_, queryErr := client.Query(q)
		assert.NoError(t, queryErr)
	})
}

func TestHeaders(t *testing.T) {
	var (
		currentHeader string
		expectedValue string
	)

	// use a test client to validate the headers are being set as expected below
	testingClient := &http.Client{Transport: &http.Transport{
		Proxy: func(request *http.Request) (*url.URL, error) {
			if val := request.Header.Get(currentHeader); val != expectedValue {
				t.Errorf("header [%s] wrong, got [%s] should be [%s]", currentHeader, val, expectedValue)
			}

			return request.URL, nil
		},
	}}

	t.Run("can set headers directly", func(t *testing.T) {
		type args struct {
			header    string
			headerOpt fauna.ClientConfigFn
		}
		tests := []struct {
			name        string
			args        args
			want        string
			expectError bool
		}{
			{
				name: "linearized should be true",
				args: args{
					headerOpt: fauna.Linearized(true),
					header:    fauna.HeaderLinearized,
				},
				want: "true",
			},
			{
				name: "timeout should be 1m",
				args: args{
					header:    fauna.HeaderTimeoutMs,
					headerOpt: fauna.QueryTimeout(time.Minute),
				},
				want: fmt.Sprintf("%d", time.Minute.Milliseconds()),
			},
			{
				name: "max contention retries should be 1",
				args: args{
					header:    fauna.HeaderMaxContentionRetries,
					headerOpt: fauna.MaxContentionRetries(1),
				},
				want: "1",
			},
			{
				name: "should have tags",
				args: args{
					header: fauna.HeaderTags,
					headerOpt: fauna.QueryTags(map[string]string{
						"hello": "world",
						"what":  "are=you,doing?",
					}),
				},
				want:        "hello=world,what=are%3Dyou%2Cdoing%3F",
				expectError: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				currentHeader = tt.args.header
				expectedValue = tt.want

				client := fauna.NewClient(
					"secret",
					fauna.URL(fauna.EndpointLocal),
					fauna.HTTPClient(testingClient),
					tt.args.headerOpt,
				)

				// running a simple query just to invoke the request
				q, _ := fauna.FQL(`Math.abs(-5.123e3)`)
				_, queryErr := client.Query(q)
				if !tt.expectError && queryErr != nil {
					t.Errorf("query failed: %s", queryErr.Error())
				}
			})
		}
	})

	t.Run("can set headers on Query", func(t *testing.T) {
		client := fauna.NewClient(
			"secret",
			fauna.URL(fauna.EndpointLocal),
			fauna.HTTPClient(testingClient),
			fauna.QueryTags(map[string]string{
				"team": "X_Men",
				"hero": "Cyclops",
			}),
		)

		currentHeader = fauna.HeaderTags
		expectedValue = "hero=Wolverine,team=X_Men"

		q, _ := fauna.FQL(`Math.abs(-5.123e3)`)
		if _, queryErr := client.Query(q, fauna.Tags(map[string]string{"hero": "Wolverine"})); queryErr != nil {
			t.Errorf("query failed: %s", queryErr.Error())
		}

		// assertion in testingClient above

		currentHeader = fauna.HeaderTraceparent
		expectedValue = "query-traceparent-id"

		q, _ = fauna.FQL(`Math.abs(-5.123e3)`)
		if _, queryErr := client.Query(q, fauna.QueryTraceparent(expectedValue)); queryErr != nil {
			t.Fatalf("failed to query with traceparent: %s", queryErr.Error())
		}
	})

	t.Run("can use convenience methods", func(t *testing.T) {
		currentHeader = fauna.HeaderLinearized
		expectedValue = "true"

		client := fauna.NewClient(
			"secret",
			fauna.URL(fauna.EndpointLocal),
			fauna.HTTPClient(testingClient),
			fauna.Linearized(true),
			fauna.QueryTimeout(time.Second*3),
			fauna.MaxContentionRetries(5),
			fauna.Context(context.Background()),
			fauna.AdditionalHeaders(map[string]string{
				"foobar":      "steve",
				currentHeader: expectedValue,
			}),
		)
		if client == nil {
			t.Errorf("failed to init client with header")
		}
	})

	t.Run("supports empty headers", func(t *testing.T) {
		client := fauna.NewClient(
			"secret",
			fauna.URL(fauna.EndpointLocal),
			fauna.AdditionalHeaders(map[string]string{
				"shouldBeEmpty": "",
			}),
		)
		if client == nil {
			t.Errorf("failed to init client with empty header")
		}
	})
}

func TestQueryTags(t *testing.T) {
	t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)
	t.Setenv(fauna.EnvFaunaSecret, "secret")

	client, clientErr := fauna.NewDefaultClient()
	if clientErr != nil {
		t.Fatalf("should be able to init default client: %s", clientErr.Error())
	}

	tags := map[string]string{
		"hello": "world",
		"what":  "areyoudoing",
	}

	// running a simple query just to invoke the request
	q, _ := fauna.FQL(`Math.abs(-5.123e3)`)
	res, queryErr := client.Query(q, fauna.Tags(tags))
	if assert.NoError(t, queryErr) {
		assert.Equal(t, tags, res.QueryTags)
	}
}

func TestErrorHandling(t *testing.T) {
	t.Run("authorization error", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "I'm a little teapot")
		t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

		client, clientErr := fauna.NewDefaultClient()
		if clientErr != nil {
			t.Fatalf("failed to init fauna.Client")
		}

		q, _ := fauna.FQL(`Math.abs(-5.123e3)`)
		_, queryErr := client.Query(q)
		if queryErr == nil {
			t.Fatalf("expected an error")
		}

		var expectedErr *fauna.AuthenticationError
		assert.ErrorAs(t, queryErr, &expectedErr)
	})

	t.Run("invalid query", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "secret")
		t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

		client, clientErr := fauna.NewDefaultClient()
		if clientErr != nil {
			t.Fatalf("failed to init fauna.Client")
		}

		q, _ := fauna.FQL(`SillyPants`)
		_, queryErr := client.Query(q)
		if queryErr == nil {
			t.Fatalf("expected an error")
		}

		var expectedErr *fauna.QueryRuntimeError
		assert.ErrorAs(t, queryErr, &expectedErr)
	})

	t.Run("service error", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "secret")
		t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

		client, clientErr := fauna.NewDefaultClient()
		if clientErr != nil {
			t.Fatalf("failed to init fauna.Client")
		}

		testCollection := "testing"

		q, _ := fauna.FQL(`Collection.create({ name: ${arg1} })`, map[string]any{"arg1": testCollection})
		_, queryErr := client.Query(q)
		if !assert.NoError(t, queryErr) {
			return
		}

		q, _ = fauna.FQL(`Collection.create({ name: ${arg1} })`, map[string]any{"arg1": testCollection})
		res, queryErr := client.Query(q)
		if queryErr == nil {
			t.Logf("response: %v", res.Data)
			t.Errorf("expected this to fail")
		} else {
			var expectedErr *fauna.QueryRuntimeError
			assert.ErrorAs(t, queryErr, &expectedErr)
		}

		q, _ = fauna.FQL(`Collection.byName(${arg1}).delete()`, map[string]any{"arg1": testCollection})
		_, queryErr = client.Query(q)
		if queryErr != nil {
			t.Fatalf("error: %s", queryErr.Error())
		}
	})
}

type Person struct {
	Name    string `fauna:"name"`
	Address string `fauna:"address"`
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
