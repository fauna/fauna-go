package fauna_test

import (
	"testing"

	"github.com/fauna/fauna-go"
	"github.com/stretchr/testify/assert"
)

type TemplateSuccessCase struct {
	given string
	wants *[]fauna.TemplatePart
}

type TemplateErrorCase struct {
	given string
	error string
}

func TestTemplate_ParseSuccess(t *testing.T) {
	testCases := []TemplateSuccessCase{
		{
			"let x = ${my_var}",
			&[]fauna.TemplatePart{
				{
					"let x = ",
					fauna.TemplateLiteral,
				},
				{
					"my_var",
					fauna.TemplateVariable,
				},
			},
		},
		{
			"let x = ${my_var}\nlet y = ${my_var}\nx * y",
			&[]fauna.TemplatePart{
				{
					"let x = ",
					fauna.TemplateLiteral,
				},
				{
					"my_var",
					fauna.TemplateVariable,
				},
				{
					"\nlet y = ",
					fauna.TemplateLiteral,
				},
				{
					"my_var",
					fauna.TemplateVariable,
				},
				{
					"\nx * y",
					fauna.TemplateLiteral,
				},
			},
		},
		{
			"${my_var} { .name }",
			&[]fauna.TemplatePart{
				{
					"my_var",
					fauna.TemplateVariable,
				},
				{
					" { .name }",
					fauna.TemplateLiteral,
				},
			},
		},
		{
			"let x = '$${not_a_var}'",
			&[]fauna.TemplatePart{
				{
					"let x = '$",
					fauna.TemplateLiteral,
				},
				{
					"{not_a_var}'",
					fauna.TemplateLiteral,
				},
			},
		},
	}

	for _, tc := range testCases {
		parsed, err := fauna.NewTemplate(tc.given).Parse()
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
		_, err := fauna.NewTemplate(tc.given).Parse()
		if assert.Error(t, err) {
			assert.EqualError(t, err, tc.error)
		}
	}
}
