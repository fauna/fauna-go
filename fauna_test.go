package fauna

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
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

func TestBasicCrudRequests(t *testing.T) {
	client, err := DefaultClient()
	assert.NoError(t, err)
	client.url = previewUrl

	coll := fmt.Sprintf("Person_%v", getRandomString(12))
	err = client.Query(fmt.Sprintf("Collection.create({ name: \"%v\" })", coll), nil, nil)
	assert.NoError(t, err)

	n := "John Smith"
	p := &Person{
		Name:    n,
		Address: "123 Range Road Houston, TX 77056",
	}
	err = client.Query(fmt.Sprintf("%v.create(%v)", coll, p.String()), nil, nil)
	assert.NoError(t, err)

	q := &Person{}
	err = client.Query(fmt.Sprintf("%v.all.firstWhere(.name == \"%v\")", coll, n), nil, q)
	assert.NoError(t, err)
	assert.Equal(t, p.Name, q.Name)

	err = client.Query(fmt.Sprintf("%v.all.firstWhere(.name == \"%v\").update({address: \"321 Rainy St Seattle, WA 98011\"})", coll, n), nil, q)
	assert.NoError(t, err)
	assert.NotEqual(t, p.Address, q.Address)

	err = client.Query(fmt.Sprintf("%v.all.firstWhere(.name == \"%v\").delete()", coll, n), nil, q)
	assert.NoError(t, err)

	err = client.Query(fmt.Sprintf("Collection.byName(\"%v\").delete()", coll), nil, nil)
	assert.NoError(t, err)
}
