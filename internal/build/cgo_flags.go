package build

import (
	"fmt"
	"strings"
	"unicode"
)

// Direct #cgo directives use go/build's quoting rules, not pkg-config's
// output convention: quotes may start within a word and adjacent segments
// concatenate. Backslashes escape the next rune, including inside quotes.
func splitCgoDirectiveFlags(input string) ([]string, error) {
	var result []string
	var word strings.Builder
	var quote rune
	escaped, present := false, false
	flush := func() {
		if present {
			result = append(result, word.String())
			word.Reset()
			present = false
		}
	}
	for _, char := range input {
		if escaped {
			word.WriteRune(char)
			escaped, present = false, true
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			} else {
				word.WriteRune(char)
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote, present = char, true
		} else if unicode.IsSpace(char) {
			flush()
		} else {
			word.WriteRune(char)
			present = true
		}
	}
	if quote != 0 || escaped {
		return nil, fmt.Errorf("invalid #cgo flags: unfinished quote or escape")
	}
	flush()
	return result, nil
}
