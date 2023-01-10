package fauna_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/fauna/fauna-go"
)

func TestDefaultClient(t *testing.T) {
	t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)
	t.Setenv(fauna.EnvFaunaSecret, "secret")

	client, err := fauna.DefaultClient()
	if err != nil {
		t.FailNow()
	}

	t.Run("should have version", func(t *testing.T) {
		if fauna.DriverVersion == "" {
			t.Errorf("driver version should not be empty")
		}
	})

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

		if res != nil {
			t.Logf("%s", res.Raw.Request.Proto)
			t.Logf("response: %s", res.Bytes)
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
		res, queryErr := client.Query(fmt.Sprintf(`%v.length`, a), map[string]interface{}{a: s}, &i)
		if queryErr != nil {
			t.Logf("response: %s", res.Bytes)
			t.Errorf("%s", queryErr.Error())
		}

		n := len(s)
		if n != i {
			t.Errorf("expected [%d] got [%d]", n, i)
		}
	})
}

func TestBasicCrudRequests(t *testing.T) {
	t.Setenv(fauna.EnvFaunaSecret, "secret")
	t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)
	client, err := fauna.DefaultClient()
	if err != nil {
		t.Errorf("%s", err.Error())
		t.FailNow()
	}

	coll := fmt.Sprintf("Person_%v", randomString(12))

	t.Run("Create a collection", func(t *testing.T) {
		res, queryErr := client.Query(`Collection.create({ name: arg1 })`, fauna.QueryArguments(fauna.QueryArg("arg1", coll)), nil)
		if queryErr != nil {
			t.Logf("error: %s\nresponse: %s", queryErr.Error(), res.Bytes)
			t.FailNow()
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
			t.Logf("error: %s\nresponse: %s", queryErr.Error(), res.Bytes)
			t.FailNow()
		}
	})

	q := Person{}
	t.Run("Query a Person", func(t *testing.T) {
		res, queryErr := client.Query(fmt.Sprintf(`%s.all.firstWhere(.name == "%s")`, coll, n), nil, &q)
		if queryErr != nil {
			t.Logf("error: %s\nresponse: %s", queryErr.Error(), res.Bytes)
			t.FailNow()
		}

		if p.Name != q.Name {
			t.Logf("response: %s", res.Bytes)
			t.Errorf("expected Name [%s] got [%s]", p.Name, q.Name)
		}
	})

	t.Run("Update a Person", func(t *testing.T) {
		res, queryErr := client.Query(fmt.Sprintf(`%v.all.firstWhere(.name == "%v").update({address: "321 Rainy St Seattle, WA 98011"})`, coll, n), nil, &q)
		if queryErr != nil {
			t.Logf("error: %s\nresponse: %s", queryErr.Error(), res.Bytes)
			t.FailNow()
		}

		if p.Address == q.Address {
			t.Errorf("expected [%s] got [%s]", p.Address, q.Address)
		}
	})

	t.Run("Delete a Person", func(t *testing.T) {
		res, queryErr := client.Query(fmt.Sprintf(`%s.all.firstWhere(.name == arg1).delete()`, coll), fauna.QueryArguments(fauna.QueryArg("arg1", p.Name)), &q)
		if queryErr != nil {
			t.Logf("error: %s\nresponse: %s", queryErr.Error(), res.Bytes)
			t.FailNow()
		}
	})

	t.Run("Delete a Collection", func(t *testing.T) {
		res, queryErr := client.Query(`Collection.byName(arg1).delete()`, fauna.QueryArguments(fauna.QueryArg("arg1", coll)), nil)
		if queryErr != nil {
			t.Logf("error: %s\nresponse: %s", queryErr.Error(), res.Bytes)
			t.FailNow()
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
				headerOpt: fauna.Timeout(time.Minute),
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
				t.Errorf("%s", queryErr.Error())
				t.FailNow()
			}
		})
	}

}

func TestErrorHandling(t *testing.T) {
	t.Run("authorization error", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "I'm a little teapot")
		t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

		client, clientErr := fauna.DefaultClient()
		if clientErr != nil {
			t.Errorf("failed to init fauna.Client")
			t.FailNow()
		}

		_, queryErr := client.Query(`Math.abs(-5.123e3)`, nil, nil)
		if queryErr == nil {
			t.Errorf("expected an error")
			t.FailNow()
		}

		if !errors.As(queryErr, &fauna.AuthenticationError{}) {
			t.Errorf("wrong type: %T", queryErr)
		}
	})

	t.Run("invalid query", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "secret")
		t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

		client, clientErr := fauna.DefaultClient()
		if clientErr != nil {
			t.Errorf("failed to init fauna.Client")
			t.FailNow()
		}

		_, queryErr := client.Query(`SillyPants`, nil, nil)
		if queryErr == nil {
			t.Errorf("expected an error")
			t.FailNow()
		}

		if !errors.As(queryErr, &fauna.QueryCheckError{}) {
			t.Errorf("wrong type: %T", queryErr)
		}
	})

	t.Run("service error", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaSecret, "secret")
		t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

		client, clientErr := fauna.DefaultClient()
		if clientErr != nil {
			t.Errorf("failed to init fauna.Client")
			t.FailNow()
		}

		testCollection := "testing"

		res, queryErr := client.Query(`Collection.create({ name: arg1 })`, fauna.QueryArguments(fauna.QueryArg("arg1", testCollection)), nil)
		if queryErr != nil {
			t.Logf("error: %s\nresponse: %s", queryErr.Error(), res.Bytes)
			t.FailNow()
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

		res, queryErr = client.Query(`Collection.byName(arg1).delete()`, fauna.QueryArguments(fauna.QueryArg("arg1", testCollection)), nil)
		if queryErr != nil {
			t.Logf("error: %s", queryErr.Error())
			t.FailNow()
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
