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
	if !assert.NoError(t, clientErr) {
		return
	}

	t.Run("basic requests", func(t *testing.T) {
		t.Run("String Length Request", func(t *testing.T) {
			s := "foo"

			query, _ := fauna.FQL(`${arg0}.length`, map[string]any{"arg0": s})

			res, queryErr := client.Query(query)
			if !assert.NoError(t, queryErr) {
				return
			}

			var i int
			marshalErr := res.Unmarshal(&i)
			if assert.NoError(t, marshalErr) {
				assert.Equal(t, len(s), i)
			}

			t.Run("response has expected stats headers", func(t *testing.T) {
				assert.Greater(t, res.Stats.ComputeOps, 0, "should have some compute ops")
				assert.GreaterOrEqual(t, res.Stats.QueryTimeMs, 0)
				assert.Zero(t, res.Stats.ContentionRetries, "should not have retried")
				assert.Zero(t, res.Stats.ReadOps, "should not have read any bytes")
				assert.Zero(t, res.Stats.WriteOps, "should not have written any bytes")
				assert.Zero(t, res.Stats.StorageBytesRead, "should not have read from storage")
				assert.Zero(t, res.Stats.StorageBytesWrite, "should not have written to storage")
			})
		})

		t.Run("Query with options", func(t *testing.T) {
			q, _ := fauna.FQL(`Math.abs(-5.123e3)`, nil)
			res, queryErr := client.Query(q, fauna.Timeout(time.Second))
			if assert.NoError(t, queryErr) {
				t.Logf("summary: %s", res.Summary)
			}
		})

		t.Run("Paginate Query", func(t *testing.T) {
			colName := "PaginationTest"
			colMod := &fauna.Module{Name: colName}

			deleteQuery, _ := fauna.FQL(`Collection.byName(${coll})?.delete()`, map[string]any{"coll": colName})
			_, deleteErr := client.Query(deleteQuery)
			if deleteErr != nil {
				t.Logf("failed to cleanup collection: %t", deleteErr)
			}

			q, _ := fauna.FQL(`Collection.create({ name: ${name} })`, map[string]any{"name": colName})
			_, createErr := client.Query(q)
			if !assert.NoError(t, createErr) {
				t.FailNow()
			}

			t.Run("a lot of items", func(t *testing.T) {
				totalTestItems := 200
				// create items
				for i := 0; i < totalTestItems; i++ {
					createCollectionQuery, createItemErr := fauna.FQL(`${mod}.create({ value: ${i} })`, map[string]any{
						"mod": colMod,
						"i":   i,
					})
					if !assert.NoError(t, createItemErr) {
						t.FailNow()
					}

					res, err := client.Query(createCollectionQuery)
					if !assert.NoError(t, err) {
						t.FailNow()
					}
					assert.NotZero(t, res.Stats.WriteOps)
				}

				// get items query
				query, queryErr := fauna.FQL(`${mod}.all()`, map[string]any{"mod": colMod})
				if !assert.NoError(t, queryErr) {
					t.FailNow()
				}

				// paginate items
				pages := 0
				itemsSeen := 0

				paginator := client.Paginate(query)
				for {
					page, err := paginator.Next()
					if !assert.NoError(t, err) || !assert.NotNil(t, page) {
						t.FailNow()
					}

					pages += 1
					itemsSeen += len(page.Data)

					t.Run("can unmarshal pages", func(t *testing.T) {
						var modItems []struct {
							Value int `fauna:"value"`
						}
						marshalErr := page.Unmarshal(&modItems)
						assert.NoError(t, marshalErr)

						assert.NotZero(t, modItems[1].Value) // use the first index to avoid zero
					})

					if !paginator.HasNext() {
						break
					}
				}

				assert.Equal(t, totalTestItems, itemsSeen)
			})

			t.Run("an incomplete page", func(t *testing.T) {
				query, queryErr := fauna.FQL(`[1,2,3,4]`, map[string]any{"mod": colMod})
				if !assert.NoError(t, queryErr) {
					t.FailNow()
				}

				// try to paginate a query that doesn't have Pages
				pages := 0
				paginator := client.Paginate(query)
				for {
					res, err := paginator.Next()
					if !assert.NoError(t, err) || !assert.NotNil(t, res) {
						t.FailNow()
					}

					pages += 1

					if !assert.NotEmpty(t, res.Data) {
						t.FailNow()
					}

					if !paginator.HasNext() {
						break
					}
				}

				assert.Equal(t, 1, pages)

			})
		})
	})
}

func TestNewClient(t *testing.T) {
	t.Run("default client", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "secret")
		_, clientErr := fauna.NewDefaultClient()
		assert.NoError(t, clientErr)
	})

	t.Run("stringify", func(t *testing.T) {
		client := fauna.NewClient("secret", fauna.DefaultTimeouts(), fauna.URL(fauna.EndpointLocal))
		assert.Equal(t, client.String(), fauna.EndpointLocal, "client toString should be equal to the endpoint to ensure we don't expose secrets")
	})

	t.Run("missing secret", func(t *testing.T) {
		_, clientErr := fauna.NewDefaultClient()
		assert.Error(t, clientErr, "should have failed due to missing secret")
	})

	t.Run("has transaction time", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "secret")
		t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

		client, clientErr := fauna.NewDefaultClient()
		if !assert.NoError(t, clientErr) {
			return
		}

		before := client.GetLastTxnTime()
		if !assert.Zero(t, before) {
			return
		}

		q, _ := fauna.FQL(`Math.abs(-5.123e3)`, nil)
		if _, queryErr := client.Query(q); !assert.NoError(t, queryErr) {
			return
		}

		first := client.GetLastTxnTime()
		assert.NotZero(t, first)

		second := client.GetLastTxnTime()
		assert.Equal(t, first, second)

		assert.NotEqual(t, before, second)
	})

	t.Run("custom HTTP client", func(t *testing.T) {
		client := fauna.NewClient(
			"secret",
			fauna.DefaultTimeouts(),
			fauna.URL(fauna.EndpointLocal),
			fauna.HTTPClient(http.DefaultClient),
		)
		q, _ := fauna.FQL(`Math.abs(-5.123e3)`, nil)
		_, queryErr := client.Query(q)
		assert.NoError(t, queryErr)
	})
}

func TestBasicCRUDRequests(t *testing.T) {
	t.Setenv(fauna.EnvFaunaSecret, "secret")
	t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)
	client, err := fauna.NewDefaultClient()
	if !assert.NoError(t, err) {
		return
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
		err := res.Unmarshal(&result)
		assert.NoError(t, err)

		assert.Equal(t, p, &result)
	})

	t.Run("Update a Person", func(t *testing.T) {
		addr := "321 Rainy St Seattle, WA 98011"
		q, _ := fauna.FQL(
			`${coll}.all().firstWhere(.name == ${name}).update({address: ${addr}})`,
			map[string]any{"coll": collMod, "name": n, "addr": addr})

		res, queryErr := client.Query(q)
		if !assert.NoError(t, queryErr) {
			return
		}

		var result Person
		err := res.Unmarshal(&result)
		assert.NoError(t, err)

		assert.Equal(t, Person{n, addr}, result)
	})

	t.Run("Delete a Person", func(t *testing.T) {
		q, _ := fauna.FQL(`${coll}.all().firstWhere(.name == ${name}).delete()`, map[string]any{"coll": collMod, "name": p.Name})
		_, queryErr := client.Query(q)
		assert.NoError(t, queryErr)
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
			assert.Equal(t, expectedValue, request.Header.Get(currentHeader))
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
					header:    fauna.HeaderQueryTimeoutMs,
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
					fauna.DefaultTimeouts(),
					fauna.URL(fauna.EndpointLocal),
					fauna.HTTPClient(testingClient),
					tt.args.headerOpt,
				)

				// running a simple query just to invoke the request
				q, _ := fauna.FQL(`Math.abs(-5.123e3)`, nil)
				_, queryErr := client.Query(q)
				if !tt.expectError {
					assert.NoError(t, queryErr)
				} else {
					assert.Error(t, queryErr)
				}
			})
		}
	})

	t.Run("can set headers on Query", func(t *testing.T) {
		client := fauna.NewClient(
			"secret",
			fauna.DefaultTimeouts(),
			fauna.URL(fauna.EndpointLocal),
			fauna.HTTPClient(testingClient),
			fauna.QueryTags(map[string]string{
				"team": "X_Men",
				"hero": "Cyclops",
			}),
		)

		currentHeader = fauna.HeaderTags
		expectedValue = "hero=Wolverine,team=X_Men"

		q, _ := fauna.FQL(`Math.abs(-5.123e3)`, nil)
		_, queryErr := client.Query(q, fauna.Tags(map[string]string{"hero": "Wolverine"}))
		assert.NoError(t, queryErr)

		// assertion in testingClient above

		currentHeader = fauna.HeaderTraceparent
		expectedValue = "query-traceparent-id"

		q, _ = fauna.FQL(`Math.abs(-5.123e3)`, nil)
		_, queryErr = client.Query(q, fauna.Traceparent(expectedValue))
		assert.NoError(t, queryErr)
	})

	t.Run("can use convenience methods", func(t *testing.T) {
		currentHeader = fauna.HeaderLinearized
		expectedValue = "true"

		client := fauna.NewClient(
			"secret",
			fauna.DefaultTimeouts(),
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

		assert.NotNil(t, client)
	})

	t.Run("supports empty headers", func(t *testing.T) {
		client := fauna.NewClient(
			"secret",
			fauna.DefaultTimeouts(),
			fauna.URL(fauna.EndpointLocal),
			fauna.AdditionalHeaders(map[string]string{
				"shouldBeEmpty": "",
			}),
		)
		assert.NotNil(t, client)
	})
}

func TestQueryTags(t *testing.T) {
	t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)
	t.Setenv(fauna.EnvFaunaSecret, "secret")

	client, clientErr := fauna.NewDefaultClient()
	if !assert.NoError(t, clientErr) {
		return
	}

	tags := map[string]string{
		"hello": "world",
		"what":  "areyoudoing",
	}

	// running a simple query just to invoke the request
	q, _ := fauna.FQL(`Math.abs(-5.123e3)`, nil)
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
		if assert.NoError(t, clientErr) {
			return
		}

		q, _ := fauna.FQL(`Math.abs(-5.123e3)`, nil)
		_, queryErr := client.Query(q)
		if assert.NoError(t, queryErr) {
			return
		}

		var expectedErr *fauna.ErrAuthentication
		assert.ErrorAs(t, queryErr, &expectedErr)
	})

	t.Run("invalid query", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "secret")
		t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

		client, clientErr := fauna.NewDefaultClient()
		if assert.NoError(t, clientErr) {
			return
		}

		q, _ := fauna.FQL(`SillyPants`, nil)
		_, queryErr := client.Query(q)

		if assert.Error(t, queryErr) {
			var expectedErr *fauna.ErrQueryRuntime
			assert.ErrorAs(t, queryErr, &expectedErr)
		}
	})

	t.Run("service error", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "secret")
		t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

		client, clientErr := fauna.NewDefaultClient()
		if !assert.NoError(t, clientErr) {
			t.FailNow()
		}

		testCollection := "testing"

		q, _ := fauna.FQL(`Collection.create({ name: ${arg1} })`, map[string]any{"arg1": testCollection})
		_, queryErr := client.Query(q)
		if !assert.NoError(t, queryErr) {
			return
		}

		q, _ = fauna.FQL(`Collection.create({ name: ${arg1} })`, map[string]any{"arg1": testCollection})
		if _, queryErr := client.Query(q); assert.Error(t, queryErr) {
			var expectedErr *fauna.ErrQueryRuntime
			assert.ErrorAs(t, queryErr, &expectedErr)
		} else {
			return
		}

		t.Run("returns a NullDoc", func(t *testing.T) {
			nullDocQuery, nullDocQueryErr := fauna.FQL(`${coll}.byId('123')`, map[string]any{"coll": &fauna.Module{Name: testCollection}})
			if !assert.NoError(t, nullDocQueryErr) {
				t.FailNow()
			}

			res, err := client.Query(nullDocQuery)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			assert.IsType(t, &fauna.NullDocument{}, res.Data)
		})

		q, _ = fauna.FQL(`Collection.byName(${arg1}).delete()`, map[string]any{"arg1": testCollection})
		_, queryErr = client.Query(q)
		assert.NoError(t, queryErr)
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
