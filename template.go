package fauna

import (
	"fmt"
	"regexp"
)

type Category string

const (
	TemplateVariable Category = "variable"
	TemplateLiteral  Category = "literal"
)

type TemplatePart struct {
	Text     string
	Category Category
}

type Template struct {
	text string
	re   *regexp.Regexp
}

func NewTemplate(text string) *Template {
	return &Template{
		text: text,
		re:   regexp.MustCompile(`\$(?:(?P<escaped>\$)|{(?P<braced>[_a-zA-Z0-9]*)}|(?P<invalid>))`),
	}
}

// Parse parses Text and returns a slice of template parts.
func (t *Template) Parse() ([]TemplatePart, error) {
	escapedIndex := t.re.SubexpIndex("escaped")
	bracedIndex := t.re.SubexpIndex("braced")
	invalidIndex := t.re.SubexpIndex("invalid")

	end := len(t.text)
	currentPosition := 0

	matches := t.re.FindAllStringSubmatch(t.text, -1)
	matchIndexes := t.re.FindAllStringSubmatchIndex(t.text, -1)
	parts := make([]TemplatePart, 0)

	for i, m := range matches {
		matchIndex := matchIndexes[i]
		invalidStartPos := matchIndex[invalidIndex*2]
		if invalidStartPos >= 0 {
			// TODO: Improve with line/column num
			return nil, fmt.Errorf("invalid placeholder in template: position %d", invalidStartPos)
		}

		matchStartPos := matchIndex[0]
		matchEndPos := matchIndex[1]
		escaped := m[escapedIndex]
		variable := m[bracedIndex]

		if currentPosition < matchStartPos {
			parts = append(parts, TemplatePart{
				Text:     t.text[currentPosition:matchStartPos] + escaped,
				Category: TemplateLiteral,
			})
		}

		if len(variable) > 0 {
			parts = append(parts, TemplatePart{
				Text:     variable,
				Category: TemplateVariable,
			})
		}

		currentPosition = matchEndPos
	}

	if currentPosition < end {
		parts = append(parts, TemplatePart{Text: t.text[currentPosition:], Category: TemplateLiteral})
	}

	return parts, nil
}
