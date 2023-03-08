package fauna_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/fauna/fauna-go"
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

			var i int
			res, queryErr := client.Query(`arg0.length`, fauna.QueryArguments(fauna.QueryArg("arg0", s)), &i)
			if queryErr != nil {
				t.Errorf("%s", queryErr.Error())
			}

			expectedProto := "HTTP/2.0"
			if res.Raw.Proto != expectedProto {
				t.Errorf("request protocol got [%s] expected [%s]", res.Raw.Proto, expectedProto)
			}

			n := len(s)
			if n != i {
				t.Errorf("expected [%d] got [%d]", n, i)
			}

			t.Run("response has expected stats headers", func(t *testing.T) {
				if res.Stats[fauna.StatsComputeOps] == 0 {
					t.Errorf("expected some compute ops")
				}

				if res.Stats[fauna.StatsQueryTimeMs] == 0 {
					t.Errorf("should have some query time")
				}

				if res.Stats[fauna.StatsContentionRetries] > 0 {
					t.Errorf("should not have any retries")
				}

				if res.Stats[fauna.StatsReadOps] > 0 || res.Stats[fauna.StatsWriteOps] > 0 {
					t.Errorf("should not have read/written any bytes")
				}

				if res.Stats[fauna.StatsStorageBytesRead] > 0 || res.Stats[fauna.StatsStorageBytesWrite] > 0 {
					t.Errorf("should not have accessed storage")
				}

				if res.FaunaBuild() == "" {
					t.Errorf("expected a fauna build")
				}

				if res.Traceparent() == "" {
					t.Errorf("should have a traceparent")
				}
			})
		})

		t.Run("Argument Request", func(t *testing.T) {
			a := "arg1"
			s := "maverick"

			var i int
			res, queryErr := client.Query(fmt.Sprintf(`%v.length`, a), fauna.QueryArguments(fauna.QueryArg(a, s)), &i)
			if queryErr != nil {
				if res != nil {
					t.Logf("response: %s", res.Bytes)
				}

				t.Fatalf("%s", queryErr)
			}

			n := len(s)
			if n != i {
				t.Errorf("expected [%d] got [%d]", n, i)
			}

			if client.GetLastTxnTime() == 0 {
				t.Errorf("last transaction time should be greater than 0")
			}
		})

		t.Run("Query with options", func(t *testing.T) {
			res, queryErr := client.Query(`Math.abs(-5.123e3)`, nil, nil, fauna.Timeout(time.Second))
			if queryErr != nil {
				t.Errorf("query failed: %s", queryErr.Error())
			}

			if res != nil {
				t.Logf("summary: %s", res.Summary)
			}
		})
	})

	t.Run("validate preview", func(t *testing.T) {
		if val, found := os.LookupEnv("FAUNA_PREVIEW_SECRET"); !found {
			t.Skip()
		} else {
			t.Setenv("FAUNA_SECRET", val)
			t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointPreview)

			previewClient, previewClientErr := fauna.NewDefaultClient()
			if previewClientErr != nil {
				t.Errorf("failed to init preview client: %v", clientErr.Error())
				t.Fail()
			}

			_, queryErr := previewClient.Query(`Math.abs(-5.123e3)`, nil, nil)
			if queryErr != nil {
				t.Errorf("query env preview failed: %v", clientErr.Error())
			}
		}
	})

	t.Run("validate query args", func(t *testing.T) {
		key := "key"
		value := "value"
		items := fauna.QueryArguments(fauna.QueryArg(key, value))
		if v := items[key]; v != value {
			t.Logf("expected [%v] got [%v]", value, v)
		}
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

		_, queryErr := client.Query(`Math.abs(-5.123e3)`, nil, nil)
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
		res, queryErr := client.Query(`Math.abs(-5.123e3)`, nil, nil)
		if queryErr != nil {
			t.Errorf("failed to query: %s", queryErr.Error())
		}

		expectedProto := "HTTP/1.1"
		if res.Raw.Proto != expectedProto {
			t.Errorf("expected protocol: %s got %s", expectedProto, res.Raw.Proto)
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

	t.Run("Create a collection", func(t *testing.T) {
		res, queryErr := client.Query(`Collection.create({ name: arg1 })`, fauna.QueryArguments(fauna.QueryArg("arg1", coll)), nil)
		if queryErr != nil {
			t.Fatalf("error: %s\nresponse: %s", queryErr.Error(), res.Bytes)
		}
	})

	n := "John Smith"
	p := &Person{
		Name:    n,
		Address: "123 Range Road Houston, TX 77056",
	}

	t.Run("Create a Person", func(t *testing.T) {
		res, queryErr := client.Query(fmt.Sprintf(`%s.create(%s)`, coll, p.String()), nil, nil)
		if queryErr != nil {
			t.Fatalf("error: %s\nresponse: %s", queryErr.Error(), res.Bytes)
		}
	})

	q := Person{}
	t.Run("Query a Person", func(t *testing.T) {
		res, queryErr := client.Query(fmt.Sprintf(`%s.all.firstWhere(.name == "%s")`, coll, n), nil, &q)
		if queryErr != nil {
			t.Fatalf("error: %s\nresponse: %s", queryErr.Error(), res.Bytes)
		}

		if res != nil {
			t.Logf("response: %s", res.Bytes)
		}

		if p.Name != q.Name {
			t.Errorf("expected Name [%s] got [%s]", p.Name, q.Name)
		}
	})

	t.Run("Update a Person", func(t *testing.T) {
		res, queryErr := client.Query(fmt.Sprintf(`%v.all.firstWhere(.name == "%v").update({address: "321 Rainy St Seattle, WA 98011"})`, coll, n), nil, &q)
		if queryErr != nil {
			t.Fatalf("error: %s\nresponse: %s", queryErr.Error(), res.Bytes)
		}

		if p.Address == q.Address {
			t.Errorf("expected [%s] got [%s]", p.Address, q.Address)
		}
	})

	t.Run("Delete a Person", func(t *testing.T) {
		res, queryErr := client.Query(fmt.Sprintf(`%s.all.firstWhere(.name == arg1).delete()`, coll), fauna.QueryArguments(fauna.QueryArg("arg1", p.Name)), &q)
		if queryErr != nil {
			t.Fatalf("error: %s\nresponse: %s", queryErr.Error(), res.Bytes)
		}
	})

	t.Run("Delete a Collection", func(t *testing.T) {
		res, queryErr := client.Query(`Collection.byName(arg1).delete()`, fauna.QueryArguments(fauna.QueryArg("arg1", coll)), nil)
		if queryErr != nil {
			t.Fatalf("error: %s\nresponse: %s", queryErr.Error(), res.Bytes)
		}
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
				_, queryErr := client.Query(`Math.abs(-5.123e3)`, nil, nil)
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

		if _, queryErr := client.Query(`Math.abs(-5.123e3)`, nil, nil, fauna.Tags(map[string]string{"hero": "Wolverine"})); queryErr != nil {
			t.Errorf("query failed: %s", queryErr.Error())
		}

		// assertion in testingClient above

		currentHeader = fauna.HeaderTraceparent
		expectedValue = "query-traceparent-id"

		if _, queryErr := client.Query(`Math.abs(-5.123e3)`, nil, nil, fauna.QueryTraceparent(expectedValue)); queryErr != nil {
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
			fauna.TypeChecking(true),
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

func TestErrorHandling(t *testing.T) {
	t.Run("authorization error", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "I'm a little teapot")
		t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

		client, clientErr := fauna.NewDefaultClient()
		if clientErr != nil {
			t.Fatalf("failed to init fauna.Client")
		}

		_, queryErr := client.Query(`Math.abs(-5.123e3)`, nil, nil)
		if queryErr == nil {
			t.Fatalf("expected an error")
		}

		if !errors.As(queryErr, &fauna.AuthenticationError{}) {
			t.Errorf("wrong type: %T", queryErr)
		}
	})

	t.Run("invalid query", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "secret")
		t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

		client, clientErr := fauna.NewDefaultClient()
		if clientErr != nil {
			t.Fatalf("failed to init fauna.Client")
		}

		_, queryErr := client.Query(`SillyPants`, nil, nil)
		if queryErr == nil {
			t.Fatalf("expected an error")
		}

		if !errors.As(queryErr, &fauna.QueryRuntimeError{}) {
			t.Errorf("wrong type: %T", queryErr)
		}
	})

	t.Run("service error", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "secret")
		t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

		client, clientErr := fauna.NewDefaultClient()
		if clientErr != nil {
			t.Fatalf("failed to init fauna.Client")
		}

		testCollection := "testing"

		res, queryErr := client.Query(`Collection.create({ name: arg1 })`, fauna.QueryArguments(fauna.QueryArg("arg1", testCollection)), nil)
		if queryErr != nil {
			t.Fatalf("error: %s\nresponse: %s", queryErr.Error(), res.Bytes)
		}

		res, queryErr = client.Query(`Collection.create({ name: arg1 })`, fauna.QueryArguments(fauna.QueryArg("arg1", testCollection)), nil)
		if queryErr == nil {
			t.Logf("response: %v", res.Data)
			t.Errorf("expected this to fail")
		} else {
			if !errors.As(queryErr, &fauna.QueryRuntimeError{}) {
				t.Errorf("wrong type: %T", queryErr)
			}
		}

		t.Logf("status: %d\nbody: %s", res.Raw.StatusCode, res.Bytes)

		_, queryErr = client.Query(`Collection.byName(arg1).delete()`, fauna.QueryArguments(fauna.QueryArg("arg1", testCollection)), nil)
		if queryErr != nil {
			t.Fatalf("error: %s", queryErr.Error())
		}
	})
}

type Person struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func (p *Person) String() string {
	j, e := json.Marshal(p)
	if e != nil {
		return ""
	}
	return string(j)
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
