package fauna

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStringLengthRequest(t *testing.T) {
	s := "foo"
	q := fmt.Sprintf("\"%v\".length", s)
	client := DefaultClient()
	client.url = previewUrl
	req := NewRequest(q)
	res := Query[int](client, req)
	assert.Equal(t, len(s), res.Data)
}
