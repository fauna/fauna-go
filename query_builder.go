package fauna

import (
	"errors"
	"fmt"
)

type Fragment interface {
	Get() any
}

type ValueFragment struct {
	value any
}

func (vf *ValueFragment) Get() any {
	return vf.value
}

func NewValueFragment(value any) *ValueFragment {
	return &ValueFragment{value}
}

type LiteralFragment struct {
	value string
}

func (lf *LiteralFragment) Get() any {
	return lf.value
}

func NewLiteralFragment(value string) *LiteralFragment {
	return &LiteralFragment{value}
}

type QueryInterpolation struct {
	Fragments []Fragment
}

func FQL(query string, args map[string]any) (*QueryInterpolation, error) {
	template := NewTemplate(query)
	parts, err := template.Parse()

	if err != nil {
		return nil, err
	}

	fragments := make([]Fragment, 0)
	for _, part := range parts {

		switch category := part.Category; category {
		case TemplateLiteral:
			fragments = append(fragments, NewLiteralFragment(part.Text))

		case TemplateVariable:
			if args == nil {
				return nil, errors.New("found template variable, but args is nil")
			}

			arg, ok := args[part.Text]

			if ok {
				fragments = append(fragments, NewValueFragment(arg))
			} else {
				return nil, fmt.Errorf("template variable %s not found in args", part.Text)
			}

		}
	}

	return &QueryInterpolation{Fragments: fragments}, nil
}
