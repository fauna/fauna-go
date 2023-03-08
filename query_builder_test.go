package fauna_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/fauna/fauna-go"
)

type fqlSuccessCase struct {
	testName string
	query    string
	args     map[string]interface{}
	wants    fauna.QueryInterpolationBuilder
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
			"simple literal case",
			"let x = 11",
			nil,
			fauna.QueryInterpolationBuilder{
				Fragments: []fauna.Fragment{fauna.NewLiteralFragment("let x = 11")},
			},
		},
		{
			"simple literal case with brace",
			"let x = { y: 11 }",
			nil,
			fauna.QueryInterpolationBuilder{
				Fragments: []fauna.Fragment{fauna.NewLiteralFragment("let x = { y: 11 }")},
			},
		},
		{
			"template variable and fauna variable",
			"let age = ${n1}\n\"Alice is #{age} years old.\"",
			map[string]interface{}{"n1": 5},
			fauna.QueryInterpolationBuilder{
				Fragments: []fauna.Fragment{
					fauna.NewLiteralFragment("let age = "),
					fauna.NewValueFragment(5),
					fauna.NewLiteralFragment("\n\"Alice is #{age} years old.\""),
				},
			},
		},
		{
			"template variable",
			"let x = ${my_var}",
			map[string]interface{}{"my_var": testDino},
			fauna.QueryInterpolationBuilder{
				Fragments: []fauna.Fragment{
					fauna.NewLiteralFragment("let x = "),
					fauna.NewValueFragment(testDino),
				},
			},
		},
		{
			"query variable",
			"${inner}\nx { .name }",
			map[string]interface{}{
				"inner": testInnerDino,
			},
			fauna.QueryInterpolationBuilder{
				Fragments: []fauna.Fragment{
					fauna.NewValueFragment(testInnerDino),
					fauna.NewLiteralFragment("\nx { .name }"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			q, err := fauna.FQL(tc.query, tc.args)

			if err != nil {
				t.Fatalf("error constructing query: %s", err)
			}

			if err != nil {
				t.Fatalf("error rendering query: %s", err)
			}

			if !buildersAreEqual(tc.wants, *q) {
				t.Errorf("(%s) expected %q but got %q", tc.testName, tc.wants, *q)
			}
		})
	}
}

func buildersAreEqual(wants fauna.QueryInterpolationBuilder, test fauna.QueryInterpolationBuilder) bool {
	isEqual := true
	for i, wantsFrag := range wants.Fragments {
		testFrag := test.Fragments[i]

		switch typedFrag := wantsFrag.Get().(type) {
		case fauna.QueryInterpolationBuilder:
			isEqual = isEqual && buildersAreEqual(typedFrag, testFrag.Get().(fauna.QueryInterpolationBuilder))
		}
		isEqual = isEqual && reflect.DeepEqual(wantsFrag.Get(), testFrag.Get())
	}

	return isEqual && len(wants.Fragments) == len(test.Fragments)
}