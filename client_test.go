package fauna_test

import (
	"fmt"
	"testing"

	"github.com/fauna/fauna-go"
)

func TestDefaultClient(t *testing.T) {
	t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointPreview)
	client, err := fauna.DefaultClient()
	if err != nil {
		t.FailNow()
	}

	t.Run("String Length Request", func(t *testing.T) {
		s := "foo"

		var i int
		if queryErr := client.Query(fmt.Sprintf(`"%v".length`, s), nil, &i); queryErr != nil {
			t.FailNow()
		}

		if len(s) != i {
			t.Fail()
		}
	})

	t.Run("Argument Request", func(t *testing.T) {
		a := "arg1"
		s := "maverick"

		var i int
		if queryErr := client.Query(fmt.Sprintf(`%v.length`, a), map[string]interface{}{a: s}, &i); queryErr != nil {
			t.FailNow()
		}

		if len(s) != i {
			t.Fail()
		}
	})

	t.Run("unauthorized client", func(t *testing.T) {
		t.Setenv(fauna.EnvFaunaKey, "I'm a little tea pot")
		failClient, clientErr := fauna.DefaultClient()
		if clientErr != nil {
			t.FailNow()
		}

		if queryErr := failClient.Query("", nil, nil); queryErr == nil {
			t.FailNow()
		}
	})
}
