package fauna_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/fauna/fauna-go"
)

func TestDefaultClient(t *testing.T) {
	t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)
	t.Setenv(fauna.EnvFaunaKey, "secret")

	client, err := fauna.DefaultClient()
	if err != nil {
		t.FailNow()
	}

	t.Run("String Length Request", func(t *testing.T) {
		s := "foo"

		var i int
		res, queryErr := client.Query(fmt.Sprintf(`"%v".length`, s), nil, &i)
		if queryErr != nil {
			t.FailNow()
		}

		t.Logf("result: %s", res.Bytes)

		if len(s) != i {
			t.Fail()
		}
	})

	t.Run("Argument Request", func(t *testing.T) {
		a := "arg1"
		s := "maverick"

		var i int
		res, queryErr := client.Query(fmt.Sprintf(`%v.length`, a), map[string]interface{}{a: s}, &i)
		if queryErr != nil {
			t.FailNow()
		}

		t.Logf("result: %s", res.Bytes)

		if len(s) != i {
			t.Fail()
		}
	})
}

func Test_UnauthorizedClient(t *testing.T) {
	t.Setenv(fauna.EnvFaunaKey, "I'm a little tea pot")
	t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)

	failClient, clientErr := fauna.DefaultClient()
	if clientErr != nil {
		t.FailNow()
	}

	res, queryErr := failClient.Query("", nil, nil)
	if queryErr == nil {
		t.Log("we expected an error")
		t.FailNow()
	}

	if res.Raw.StatusCode != http.StatusUnauthorized {
		t.FailNow()
	}
}
