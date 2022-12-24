package fauna_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"

	"github.com/fauna/fauna-go"
)

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
	t.Setenv(fauna.EnvFaunaEndpoint, fauna.EndpointPreview)
	client, err := fauna.DefaultClient()
	if err != nil {
		t.FailNow()
	}

	coll := fmt.Sprintf("Person_%v", randomString(12))

	t.Run("Create a collection", func(t *testing.T) {
		if queryErr := client.Query(fmt.Sprintf(`Collection.create({ name: "%v" })`, coll), nil, nil); queryErr != nil {
			t.FailNow()
		}
	})

	n := "John Smith"
	p := &Person{
		Name:    n,
		Address: "123 Range Road Houston, TX 77056",
	}
	t.Run("Create a Person", func(t *testing.T) {
		if queryErr := client.Query(fmt.Sprintf("%v.create(%v)", coll, p.String()), nil, nil); queryErr != nil {
			t.FailNow()
		}
	})

	q := Person{}
	t.Run("Query a Person", func(t *testing.T) {
		if queryErr := client.Query(fmt.Sprintf(`%v.all.firstWhere(.name == "%v")`, coll, n), nil, &q); queryErr != nil {
			t.FailNow()
		}
		if p.Name != q.Name {
			t.Fail()
		}
	})

	t.Run("Update a Person", func(t *testing.T) {
		if queryErr := client.Query(fmt.Sprintf(`%v.all.firstWhere(.name == "%v").update({address: "321 Rainy St Seattle, WA 98011"})`, coll, n), nil, &q); queryErr != nil {
			t.FailNow()
		}
		if p.Address == q.Address {
			t.Fail()
		}
	})

	t.Run("Delete a Person", func(t *testing.T) {
		if queryErr := client.Query(fmt.Sprintf(`%v.all.firstWhere(.name == "%v").delete()`, coll, n), nil, q); queryErr != nil {
			t.FailNow()
		}
	})

	t.Run("Delete a Collection", func(t *testing.T) {
		if queryErr := client.Query(fmt.Sprintf(`Collection.byName("%v").delete()`, coll), nil, nil); queryErr != nil {
			t.FailNow()
		}
	})
}
