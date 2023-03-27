package fauna

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type TemplateSuccessCase struct {
	given string
	wants *[]templatePart
}

type TemplateErrorCase struct {
	given string
	error string
}

func TestTemplate_ParseSuccess(t *testing.T) {
	testCases := []TemplateSuccessCase{
		{
			"let x = ${my_var}",
			&[]templatePart{
				{
					"let x = ",
					templateLiteral,
				},
				{
					"my_var",
					templateVariable,
				},
			},
		},
		{
			"let x = ${my_var}\nlet y = ${my_var}\nx * y",
			&[]templatePart{
				{
					"let x = ",
					templateLiteral,
				},
				{
					"my_var",
					templateVariable,
				},
				{
					"\nlet y = ",
					templateLiteral,
				},
				{
					"my_var",
					templateVariable,
				},
				{
					"\nx * y",
					templateLiteral,
				},
			},
		},
		{
			"${my_var} { .name }",
			&[]templatePart{
				{
					"my_var",
					templateVariable,
				},
				{
					" { .name }",
					templateLiteral,
				},
			},
		},
		{
			"let x = '$${not_a_var}'",
			&[]templatePart{
				{
					"let x = '$",
					templateLiteral,
				},
				{
					"{not_a_var}'",
					templateLiteral,
				},
			},
		},
	}

	for _, tc := range testCases {
		parsed, err := newTemplate(tc.given).Parse()
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, len(*tc.wants), len(parsed))

		for i, tp := range parsed {
			expected := (*tc.wants)[i]
			assert.Equal(t, expected.Text, tp.Text)
			assert.Equal(t, expected.Category, tp.Category)
		}
	}
}

func TestTemplate_ParseFail(t *testing.T) {
	testCases := []TemplateErrorCase{
		{
			"let x = ${かわいい}",
			"invalid placeholder in template: position 9",
		},
	}

	for _, tc := range testCases {
		_, err := newTemplate(tc.given).Parse()
		if assert.Error(t, err) {
			assert.EqualError(t, err, tc.error)
		}
	}
}
