package fauna

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestArrayContainsString(t *testing.T) {
	s := "Ringo"
	a := []string{"John", "Paul", s}
	b := arrayContains(a, s)
	assert.True(t, b)
}

func TestArrayNotContainsString(t *testing.T) {
	s := "Ringo"
	a := []string{"John", "Paul", "Elvis"}
	b := arrayContains(a, s)
	assert.False(t, b)
}
