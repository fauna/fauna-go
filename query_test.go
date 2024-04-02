package fauna

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fqlSuccessCase struct {
	testName string
	query    string
	args     map[string]any
	wants    *Query
}

func TestFQL(t *testing.T) {
	testDate := time.Date(2023, 2, 24, 0, 0, 0, 0, time.UTC)
	testDino := map[string]any{
		"name":      "Dino",
		"age":       0,
		"birthdate": testDate,
	}
	testInnerDino, _ := FQL("let x = ${my_var}", map[string]any{"my_var": testDino})
	testCases := []fqlSuccessCase{
		{
			"simple literal case",
			"let x = 11",
			nil,
			&Query{
				fragments: []*queryFragment{{true, "let x = 11"}},
			},
		},
		{
			"simple literal case with brace",
			"let x = { y: 11 }",
			nil,
			&Query{
				fragments: []*queryFragment{{true, "let x = { y: 11 }"}},
			},
		},
		{
			"template variable and fauna variable",
			"let age = ${n1}\n\"Alice is #{age} years old.\"",
			map[string]any{"n1": 5},
			&Query{
				fragments: []*queryFragment{
					{true, "let age = "},
					{false, 5},
					{true, "\n\"Alice is #{age} years old.\""},
				},
			},
		},
		{
			"template variable",
			"let x = ${my_var}",
			map[string]any{"my_var": testDino},
			&Query{
				fragments: []*queryFragment{
					{true, "let x = "},
					{false, testDino},
				},
			},
		},
		{
			"query variable",
			"${inner}\nx { name }",
			map[string]any{
				"inner": testInnerDino,
			},
			&Query{
				fragments: []*queryFragment{
					{false, testInnerDino},
					{true, "\nx { name }"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			if q, err := FQL(tc.query, tc.args); assert.NoError(t, err) {
				assert.Equal(t, tc.wants, q)
			}
		})
	}
}

func BenchmarkFQL(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = FQL(`${arg0}.length`, map[string]any{"arg0": "foo"})
	}
}
