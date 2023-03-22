package fauna

import (
	"errors"
	"fmt"
)

type queryFragment struct {
	literal bool
	value   any
}

type Query struct {
	fragments []*queryFragment
}

func FQL(query string, args map[string]any) (*Query, error) {
	template := NewTemplate(query)
	parts, err := template.Parse()

	if err != nil {
		return nil, err
	}

	fragments := make([]*queryFragment, 0)
	for _, part := range parts {

		switch category := part.Category; category {
		case TemplateLiteral:
			fragments = append(fragments, &queryFragment{true, part.Text})

		case TemplateVariable:
			if args == nil {
				return nil, errors.New("found template variable, but args is nil")
			}

			arg, ok := args[part.Text]

			if ok {
				fragments = append(fragments, &queryFragment{false, arg})
			} else {
				return nil, fmt.Errorf("template variable %s not found in args", part.Text)
			}

		}
	}

	return &Query{fragments: fragments}, nil
}
