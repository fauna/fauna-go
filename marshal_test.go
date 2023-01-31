package fauna_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/fauna/fauna-go"
)

func TestMarshal(t *testing.T) {
	t.Run("validate all types", func(t *testing.T) {
		type foobar struct {
			Name           string    `fauna:"name"`
			Date           time.Time `fauna:"date" faunaType:"@date"`
			Time           time.Time `fauna:"time"`
			Long           int64     `fauna:"long"`
			TaggedLong     int64     `fauna:"taggedLong" faunaType:"@long"`
			Number         int32     `fauna:"number"`
			UntaggedDouble float32   `fauna:"decimal"`
			TaggedDouble   int       `fauna:"taggedDouble" faunaType:"@double"`
			OmitMe         int32     `fauna:"-"`
			AtPlace        string    `fauna:"@place"`
			Child          struct {
				More struct {
					Silly time.Time `fauna:"working"`
				} `fauna:"more" faunaType:"@object"`
			} `fauna:"child" faunaType:"@object"`
		}

		d, _ := time.Parse("2006-01-02", "1923-05-13")
		timestamp, _ := time.Parse(time.RFC3339, "2023-01-30T16:31:24.077936-05:00")

		result, err := fauna.Marshal(foobar{
			Name:           "Hello, World",
			Date:           d,
			Time:           timestamp,
			Long:           32,
			TaggedLong:     48,
			Number:         32,
			UntaggedDouble: 4.14,
			TaggedDouble:   30,
			AtPlace:        "Home sweet home",
		})
		if err != nil {
			t.Errorf("should not have errored: %s", err.Error())
		}

		t.Logf("result: %s", result)

		resultMap := map[string]interface{}{}
		_ = json.Unmarshal(result, &resultMap)

		tests := []struct {
			name     string
			expected string
			key      string
		}{
			{
				name:     "should have a name",
				expected: "Hello, World",
				key:      "name",
			},
			{
				name:     "should have a date",
				expected: `map[@date:1923-05-13T00:00:00Z]`,
				key:      "date",
			},
			{
				name:     "should have a double",
				expected: `map[@double:4.14]`,
				key:      "decimal",
			},
			{
				name:     "should have a taggedDouble",
				expected: `map[@double:30]`,
				key:      "taggedDouble",
			},
			{
				name:     "should have a tagged struct",
				expected: `map[@object:map[more:map[@object:map[working:map[@date:0001-01-01T00:00:00Z]]]]]`,
				key:      "child",
			},
			{
				name:     "should have a tagged long",
				expected: `map[@long:48]`,
				key:      "taggedLong",
			},
			{
				name:     "should not have omitted items",
				expected: "<nil>",
				key:      "notUsed",
			},
			{
				name:     "should allow keys with @ prefix",
				expected: "Home sweet home",
				key:      "@place",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if fmt.Sprintf("%v", resultMap[tt.key]) != tt.expected {
					t.Errorf("%s failed\nexpected: %s\ngot:      %v\n", tt.key, tt.expected, resultMap[tt.key])
				}
			})
		}
	})

	t.Run("validate invalid tag", func(t *testing.T) {
		expectedError := "unsupported fauna tag [@cowbell] on struct field [Name]"

		if b, err := fauna.Marshal(struct {
			Name string `faunaType:"@cowbell"`
		}{Name: "Steve"}); err == nil {
			t.Errorf("should not have been able to marshal")
			t.Logf("result: %s", b)
		} else if err.Error() != expectedError {
			t.Errorf("unexpected error format: %s\nexpected: %s\n", err.Error(), expectedError)
		}
	})
}
