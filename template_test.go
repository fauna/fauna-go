package fauna_test

import (
	"testing"

	"github.com/fauna/fauna-go"
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

		if err != nil {
			t.Fatalf("could not parse template: %s. err: %s", tc.given, err)
		}

		if len(parsed) != len(*tc.wants) {
			t.Errorf("wants %q but got %q", tc.wants, parsed)
		}

		for i, tp := range parsed {
			expected := (*tc.wants)[i]
			if tp.Text != expected.Text && tp.Category != expected.Category {
				t.Errorf("wants %q but got %q", expected, tp)
			}
		}
	}
}

func TestTemplate_ParseFail(t *testing.T) {
	testCases := []TemplateErrorCase{
		{
			"let x = ${かわいい}",
			"Invalid placeholder in template: position 9",
		},
	}

	for _, tc := range testCases {
		parsed, err := fauna.NewTemplate(tc.given).Parse()

		if err == nil {
			t.Fatalf("expected error, but instead we parsed %s into %q", tc.given, parsed)
		}

		if err.Error() != tc.error {
			t.Errorf("Expected error `%s` but received `%s`", tc.error, err.Error())
		}
	}
}
