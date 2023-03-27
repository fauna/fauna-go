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

func FQL(query string, args ...map[string]any) (*Query, error) {
	template := newTemplate(query)
	parts, err := template.Parse()

	if err != nil {
		return nil, err
	}

	var qArgs map[string]any
	if len(args) == 1 {
		qArgs = args[0]
	} else if len(args) > 1 {
		qArgs = map[string]any{}
		for _, a := range args {
			for k, v := range a {
				qArgs[k] = v
			}
		}
	}

	fragments := make([]*queryFragment, 0)
	for _, part := range parts {

		switch category := part.Category; category {
		case templateLiteral:
			fragments = append(fragments, &queryFragment{true, part.Text})

		case templateVariable:
			if args == nil {
				return nil, errors.New("found template variable, but args is nil")
			}

			arg, ok := qArgs[part.Text]

			if ok {
				fragments = append(fragments, &queryFragment{false, arg})
			} else {
				return nil, fmt.Errorf("template variable %s not found in args", part.Text)
			}

		}
	}

	return &Query{fragments: fragments}, nil
}
