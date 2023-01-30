package fauna_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
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
			res, queryErr := client.Query(fmt.Sprintf(`"%v".length`, s), nil, &i)
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

	t.Run("invalid timeout", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaTimeout, "invalidTime")

		_, invalidErr := fauna.NewDefaultClient()
		if invalidErr != nil {
			t.Errorf("invalid: %s", invalidErr.Error())
		}
	})

	t.Run("with observer", func(t *testing.T) {
		var observerResult fauna.ObserverResult

		beforeQuery := time.Now()

		observer := func(result *fauna.ObserverResult) {
			observerResult = *result
		}

		client.SetObserver(observer)
		_, queryErr := client.Query(`Math.abs(-5.123e3)`, nil, nil)

		afterQuery := time.Now()

		if queryErr != nil {
			t.Fatalf("failed to invoke query: %s", queryErr.Error())
		}

		if observerResult.HttpResponse == nil {
			t.Errorf("should have an HTTP Response")
		}

		if observerResult.FaunaResponse == nil {
			t.Errorf("should have a Fauna Response")
		}

		if observerResult.Error != nil {
			t.Errorf("fauna error should be nil")
		}

		if !observerResult.TimeStart.After(beforeQuery) || !observerResult.TimeStart.Before(afterQuery) {
			t.Errorf("start time is wrong: got [%s] expected after [%s] and before [%s]",
				observerResult.TimeStart.Format(time.RFC3339Nano),
				beforeQuery.Format(time.RFC3339Nano),
				afterQuery.Format(time.RFC3339Nano),
			)
		}

		if !afterQuery.After(observerResult.TimeEnd) {
			t.Errorf(
				"end time is wrong: got [%s] expected after [%s]\ndiff: %s",
				observerResult.TimeEnd.Format(time.RFC3339Nano),
				afterQuery.Format(time.RFC3339Nano),
				afterQuery.Sub(observerResult.TimeEnd),
			)
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

	t.Run("missing secret", func(t *testing.T) {
		_, clientErr := fauna.NewDefaultClient()
		if clientErr == nil {
			t.Errorf("should have failed due to missing secret")
		}
	})

	t.Run("custom timeout", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "secret")
		t.Setenv(fauna.EnvFaunaTimeout, "3s")

		_, clientErr := fauna.NewDefaultClient()
		if clientErr != nil {
			t.Errorf("should be able to init a client with a custom timeout: %s", clientErr.Error())
		}
	})

	t.Run("custom logger", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaVerboseDebugEnabled, "true")
		b := bytes.NewBuffer(nil)

		client := fauna.NewClient("secret",
			fauna.URL(fauna.EndpointLocal),
			fauna.Logger(log.New(b, t.Name(), 0)),
		)
		_, queryErr := client.Query(`dbg("sup")`, nil, nil)
		if queryErr != nil {
			t.Fatalf("query error: %s", queryErr.Error())
		}

		if b.String() == "" {
			t.Errorf("expected logger to have contents")
		}
	})

	t.Run("disable type checking", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "secret")
		t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

		t.Run("at client", func(t *testing.T) {
			t.Setenv(fauna.EnvFaunaTypeCheckEnabled, "false")

		})

		t.Run("at request", func(t *testing.T) {
			client, clientErr := fauna.NewDefaultClient()
			if clientErr != nil {
				t.Fatalf("should be able to init client: %s", clientErr.Error())
			}

			_, queryErr := client.Query(`Math.abs(-5.123e3)`, nil, nil, fauna.QueryTypeChecking(false))
			if queryErr != nil {
				t.Fatalf("should be able to query without type checking: %s", queryErr)
			}
		})
	})

	t.Run("verbose enabled", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "secret")
		t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

		t.Run("at client", func(t *testing.T) {
			t.Setenv(fauna.EnvFaunaVerboseDebugEnabled, "true")

			b := bytes.NewBuffer(nil)
			log.SetOutput(b)

			client, clientErr := fauna.NewDefaultClient()
			if clientErr != nil {
				t.Fatalf("should be able to init client: %s", clientErr.Error())
			}

			res, queryErr := client.Query(`Math.abs(-5.123e3)`, nil, nil)
			if queryErr != nil {
				t.Fatalf("should be able to query without type checking: %s", queryErr)
			}

			logBuf := b.String()
			t.Logf("response: %s", res.Bytes)

			if !strings.Contains(logBuf, "REQUEST:") {
				t.Errorf("Expected request output\nbuffer: %s\n", logBuf)
			}

			if !strings.Contains(logBuf, "RESPONSE:") {
				t.Errorf("Expected response output\nbuffer: %s\n", logBuf)
			}
		})

		t.Run("can bump last txn time", func(t *testing.T) {
			t.Setenv(fauna.EnvFaunaSecret, "secret")
			t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

			client, clientErr := fauna.NewDefaultClient()
			if clientErr != nil {
				t.Fatalf("failed to init client: %s", clientErr.Error())
			}

			_, queryErr := client.Query(`Math.abs(-5.123e3)`, nil, nil)
			if queryErr != nil {
				t.Fatalf("failed to query: %s", queryErr.Error())
			}

			txtTimeBeforeBump := client.GetLastTxnTime()

			time.Sleep(time.Millisecond * 250)

			bumpTxnTimeErr := client.SetLastTxnTime(time.Now())
			if bumpTxnTimeErr != nil {
				t.Fatalf("failed to bump txn time: %s", bumpTxnTimeErr.Error())
			}

			if txtTimeBeforeBump == client.GetLastTxnTime() {
				t.Errorf("last txn time has not changed")
			}

			_, secondQueryErr := client.Query(`Math.abs(-5.123e3)`, nil, nil)
			if secondQueryErr != nil {
				t.Fatalf("second query failed: %s", secondQueryErr.Error())
			}

			badTxnTimeErr := client.SetLastTxnTime(time.Now().Add(-(time.Hour * 1)))
			if badTxnTimeErr == nil {
				t.Errorf("setting the txn time backwards should have failed")
			}
		})
	})

	t.Run("disable last transaction time", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "secret")
		t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

		t.Run("at client", func(t *testing.T) {
			t.Setenv(fauna.EnvFaunaTrackTxnTimeEnabled, "false")

			client, clientErr := fauna.NewDefaultClient()
			if clientErr != nil {
				t.Errorf("should be able to init a client without type checking: %s", clientErr.Error())
			}

			first := client.GetLastTxnTime()
			if first != 0 {
				t.Errorf("shouldn't have a transaction time")
			}

			_, queryErr := client.Query(`Math.abs(-5.123e3)`, nil, nil)
			if queryErr != nil {
				t.Fatalf("query shouldn't error: %s", queryErr.Error())
			}

			if after := client.GetLastTxnTime(); after != 0 {
				t.Errorf("last transaction time should still be 0, got: %d", after)
			}
		})

		t.Run("at request", func(t *testing.T) {
			client, clientErr := fauna.NewDefaultClient()
			if clientErr != nil {
				t.Fatalf("should be able to init a client: %s", clientErr.Error())
			}

			first := client.GetLastTxnTime()
			if first != 0 {
				t.Fatalf("shouldn't have a transaction time")
			}

			_, queryErr := client.Query(`Math.abs(-5.123e3)`, nil, nil)
			if queryErr != nil {
				t.Fatalf("query shouldn't error: %s", queryErr.Error())
			}

			before := client.GetLastTxnTime()
			if before == 0 {
				t.Errorf("should have a last transaction time greater than 0, got: %d", before)
			}

			_, queryErr = client.Query(`Math.abs(-5.123e3)`, nil, nil, fauna.QueryTxnTime(false))
			if queryErr != nil {
				t.Fatalf("query shouldn't error: %s", queryErr.Error())
			}

			after := client.GetLastTxnTime()
			if before != after {
				t.Errorf("transaction time not have changed, before [%d] after [%d]", before, after)
			}
		})
	})

	t.Run("with observer", func(t *testing.T) {
		client := fauna.NewClient(
			"secret",
			fauna.URL(fauna.EndpointLocal),
			fauna.Observer(func(result *fauna.ObserverResult) {}),
		)
		if client == nil {
			t.Fatalf("failed to init client with observer")
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
			name string
			args args
			want string
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
				if queryErr != nil {
					t.Fatalf("%s", queryErr.Error())
				}
			})
		}
	})

	t.Run("can use convenience methods", func(t *testing.T) {
		currentHeader = fauna.HeaderLinearized
		expectedValue = "true"

		client := fauna.NewClient(
			"secret", fauna.URL(fauna.EndpointLocal),
			fauna.HTTPClient(testingClient),
			fauna.Linearized(true),
			fauna.QueryTimeout(time.Second*3),
			fauna.MaxContentionRetries(5),
			fauna.Context(context.Background()),
			fauna.TypeChecking(true),
			fauna.Headers(map[string]string{
				"foobar": "steve",
			}),
		)
		client.SetHeader(currentHeader, expectedValue)
	})

	t.Run("supports empty headers", func(t *testing.T) {
		client := fauna.NewClient("secret", fauna.URL(fauna.EndpointLocal))
		client.SetHeader("steve", "empty")
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

		if !errors.As(queryErr, &fauna.QueryCheckError{}) {
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
			if !errors.As(queryErr, &fauna.ServiceInternalError{}) {
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

func TestConcurrentRequests(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		txnTime := time.Now()

		w.Header().Set(fauna.HeaderContentType, "application/json")
		w.Header().Set(fauna.HeaderTxnTime, fmt.Sprintf("%d", txnTime.UnixMicro()))

		_, _ = fmt.Fprintf(w, `{"data": { "hello": "world" }, "error": {"code": "", "message": ""}, "summary": "", "txn_time": "%s"}`, txnTime.Format("2006-01-02T15:04:05.000Z"))
	}))
	ts.EnableHTTP2 = true
	defer ts.Close()

	client := fauna.NewClient("", fauna.URL(ts.URL))

	iterations := 100

	var wg sync.WaitGroup
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {

		go func() {
			defer wg.Done()

			_, queryErr := client.Query(`Now()`, nil, nil)
			if queryErr != nil {
				t.Errorf("failed to query: %s", queryErr.Error())
			}

			_ = client.SetLastTxnTime(time.Now())
		}()
	}

	wg.Wait()
}
