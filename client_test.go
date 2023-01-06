package fauna_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
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
			t.Logf("response: %s", res.Bytes)
			t.Errorf("%s", queryErr.Error())
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

func TestBasicCrudRequests(t *testing.T) {
	t.Setenv(fauna.EnvFaunaKey, "secret")
	t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointLocal)
	client, err := fauna.DefaultClient()
	if err != nil {
		t.Errorf("%s", err.Error())
		t.FailNow()
	}

	coll := fmt.Sprintf("Person_%v", randomString(12))

	t.Run("Create a collection", func(t *testing.T) {
		res, queryErr := client.Query(`Collection.create({ name: arg1 })`, map[string]interface{}{"arg1": coll}, nil)
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
		res, queryErr := client.Query(fmt.Sprintf(`%s.all.firstWhere(.name == arg1).delete()`, coll), map[string]interface{}{"arg1": p.Name}, &q)
		if queryErr != nil {
			t.Logf("error: %s\nresponse: %s", queryErr.Error(), res.Bytes)
			t.FailNow()
		}
	})

	t.Run("Delete a Collection", func(t *testing.T) {
		res, queryErr := client.Query(`Collection.byName(arg1).delete()`, map[string]interface{}{"arg1": coll}, nil)
		if queryErr != nil {
			t.Logf("error: %s\nresponse: %s", queryErr.Error(), res.Bytes)
			t.FailNow()
		}
	})
}
