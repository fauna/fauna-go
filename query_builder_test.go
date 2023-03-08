package fauna_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/fauna/fauna-go"
)

type fqlSuccessCase struct {
	query string
	args  map[string]interface{}
	wants map[string][]interface{}
}

func TestFQL(t *testing.T) {
	testDate := time.Date(2023, 2, 24, 0, 0, 0, 0, time.UTC)
	testDino := map[string]interface{}{
		"name":      "Dino",
		"age":       0,
		"birthdate": testDate,
	}
	testInnerDino, _ := fauna.FQL("let x = ${my_var}", map[string]interface{}{"my_var": testDino})
	testCases := []fqlSuccessCase{
		{
			"let x = 11",
			nil,
			map[string][]interface{}{
				"fql": {"let x = 11"},
			},
		},
		{
			"let x = { y: 11 }",
			nil,
			map[string][]interface{}{
				"fql": {"let x = { y: 11 }"},
			},
		},
		{
			"let age = ${n1}\n\"Alice is #{age} years old.\"",
			map[string]interface{}{"n1": 5},
			map[string][]interface{}{
				"fql": {"let age = ", map[string]interface{}{"value": 5}, "\n\"Alice is #{age} years old.\""},
			},
		},
		{
			"let x = ${my_var}",
			map[string]interface{}{"my_var": testDino},
			map[string][]interface{}{
				"fql": {"let x = ", map[string]interface{}{
					"value": testDino},
				},
			},
		},
		{
			"${inner}\nx { .name }",
			map[string]interface{}{
				"inner": *testInnerDino,
			},
			map[string][]interface{}{
				"fql": {
					map[string][]interface{}{
						"fql": {"let x = ", map[string]interface{}{
							"value": testDino},
						},
					},
					"\nx { .name }",
				},
			},
		},
	}

	for _, tc := range testCases {
		q, err := fauna.FQL(tc.query, tc.args)

		if err != nil {
			t.Fatalf("error constructing query: %s", err)
		}

		rendered, err := q.ToQuery()

		if err != nil {
			t.Fatalf("error rendering query: %s", err)
		}

		if !reflect.DeepEqual(tc.wants, rendered) {
			t.Errorf("expected %q but got %q", tc.wants, rendered)
		}
	}
}
