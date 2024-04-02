package fauna

import (
	"errors"
	"fmt"
)

type queryFragment struct {
	literal bool
	value   any
}

// Query represents a query to be sent to Fauna.
type Query struct {
	fragments []*queryFragment
}

// FQL creates a [fauna.Query] from an FQL string and set of arguments.
//
// args are optional. If provided their keys must match with `${name}` sigils
// in the query. FQL `${value} + 1` must have an argument called "value" in the
// args map.
//
// The values of args can be any type, including [fauna.Query] to allow for
// query composition.
func FQL(query string, args map[string]any) (*Query, error) {
	parts, err := parseTemplate(query)
	if err != nil {
		return nil, err
	}

	fragments := make([]*queryFragment, 0, len(parts))
	for _, part := range parts {
		switch category := part.Category; category {
		case templateLiteral:
			fragments = append(fragments, &queryFragment{true, part.Text})

		case templateVariable:
			if args == nil {
				return nil, errors.New("found template variable, but args is nil")
			}

			if arg, ok := args[part.Text]; ok {
				fragments = append(fragments, &queryFragment{false, arg})
			} else {
				return nil, fmt.Errorf("template variable %s not found in args", part.Text)
			}

		}
	}

	return &Query{fragments: fragments}, nil
}
