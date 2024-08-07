package fauna

import (
	"fmt"
	"regexp"
)

type templateCategory string

const (
	templateVariable templateCategory = "variable"
	templateLiteral  templateCategory = "literal"
)

var (
	templateRegex = regexp.MustCompile(`\$(?:(?P<escaped>\$)|{(?P<braced>[_a-zA-Z0-9]*)}|(?P<invalid>))`)
	escapedIndex  = templateRegex.SubexpIndex("escaped")
	bracedIndex   = templateRegex.SubexpIndex("braced")
	invalidIndex  = templateRegex.SubexpIndex("invalid")
)

type templatePart struct {
	Text     string
	Category templateCategory
}

// Parse parses Text and returns a slice of template parts.
func parseTemplate(text string) ([]templatePart, error) {
	end := len(text)
	currentPosition := 0

	matches := templateRegex.FindAllStringSubmatch(text, -1)
	matchIndexes := templateRegex.FindAllStringSubmatchIndex(text, -1)
	parts := make([]templatePart, 0)

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
			parts = append(parts, templatePart{
				Text:     text[currentPosition:matchStartPos] + escaped,
				Category: templateLiteral,
			})
		}

		if len(variable) > 0 {
			parts = append(parts, templatePart{
				Text:     variable,
				Category: templateVariable,
			})
		}

		currentPosition = matchEndPos
	}

	if currentPosition < end {
		parts = append(parts, templatePart{Text: text[currentPosition:], Category: templateLiteral})
	}

	return parts, nil
}
