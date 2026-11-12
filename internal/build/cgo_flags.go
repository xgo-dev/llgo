package build

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

// Match Go's source-directory expansion and relative -I/-L handling without
// imposing another flag safety policy. The caller limits this to direct #cgo
// directives; pkg-config's already-resolved output never enters this helper.
func normalizeCgoDirectivePaths(args []string, sourceDir string) {
	for i, arg := range args {
		args[i] = strings.ReplaceAll(arg, "${SRCDIR}", filepath.ToSlash(sourceDir))
	}
	nextPath := false
	for i, arg := range args {
		if nextPath {
			if !filepath.IsAbs(arg) {
				args[i] = filepath.Join(sourceDir, arg)
			}
			nextPath = false
		} else if strings.HasPrefix(arg, "-I") || strings.HasPrefix(arg, "-L") {
			if len(arg) == 2 {
				nextPath = true
			} else if !filepath.IsAbs(arg[2:]) {
				args[i] = arg[:2] + filepath.Join(sourceDir, arg[2:])
			}
		}
	}
}

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
