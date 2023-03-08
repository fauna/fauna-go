package fauna

import (
	"errors"
	"fmt"
)

type Fragment interface {
	render() (interface{}, error)
}

type ValueFragment struct {
	val interface{}
}

func (vf ValueFragment) render() (interface{}, error) {
	// TODO: Encode the value into tagged format
	return map[string]interface{}{
		"value": vf.val,
	}, nil
}

type LiteralFragment struct {
	val string
}

func (lf LiteralFragment) render() (interface{}, error) {
	return lf.val, nil
}

type QueryFragment struct {
	builder QueryBuilder
}

func (qf QueryFragment) render() (interface{}, error) {
	return qf.builder.ToQuery()
}

type QueryBuilder struct {
	fragments []Fragment
}

func (qb *QueryBuilder) ToQuery() (map[string][]interface{}, error) {
	rendered := make([]interface{}, len(qb.fragments))

	for i, f := range qb.fragments {
		renderedFrag, err := f.render()
		if err != nil {
			return nil, err
		}

		rendered[i] = renderedFrag
	}

	return map[string][]interface{}{
		"fql": rendered,
	}, nil
}

func FQL(query string, args map[string]interface{}) (*QueryBuilder, error) {
	template := NewTemplate(query)
	parts, err := template.Parse()

	if err != nil {
		return nil, err
	}

	fragments := make([]Fragment, 0)
	for _, part := range parts {

		switch category := part.Category; category {
		case TemplateLiteral:
			fragments = append(fragments, LiteralFragment{part.Text})

		case TemplateVariable:
			if args == nil {
				return nil, errors.New("found template variable, but args is nil")
			}

			arg, ok := args[part.Text]

			if ok {
				switch typed := arg.(type) {
				case QueryBuilder:
					fragments = append(fragments, QueryFragment{builder: typed})

				default:
					fragments = append(fragments, ValueFragment{val: typed})
				}

			} else {
				return nil, fmt.Errorf("template variable %s not found in args", part.Text)
			}

		}
	}

	return &QueryBuilder{fragments: fragments}, nil
}
