package fauna

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStringLengthRequest(t *testing.T) {
	client, err := DefaultClient()
	assert.NoError(t, err)
	client.url = previewUrl
	s := "foo"
	q := fmt.Sprintf("\"%v\".length", s)
	var i int
	err = client.Query(q, nil, &i)
	assert.NoError(t, err)
	assert.Equal(t, len(s), i)
}

func TestStringLengthArgumentRequest(t *testing.T) {
	client, err := DefaultClient()
	assert.NoError(t, err)
	client.url = previewUrl
	a := "arg1"
	s := "maverick"
	q := fmt.Sprintf("%v.length", a)
	var i int
	err = client.Query(q, map[string]string{a: s}, &i)
	assert.NoError(t, err)
	assert.Equal(t, len(s), i)
}
