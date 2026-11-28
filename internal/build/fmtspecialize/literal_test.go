package fmtspecialize

import (
	"strings"
	"testing"
)

func TestLiteralFormat(t *testing.T) {
	for _, format := range []string{"", "hello", "你好\x00世界", "%%", "a%%b%%%%c", "\xff%%"} {
		got, ok := literalFormat(format)
		if !ok || got != strings.ReplaceAll(format, "%%", "%") {
			t.Fatalf("literalFormat(%q) = %q, %v", format, got, ok)
		}
	}
	for _, format := range []string{"%", "%%%", "%s", "%[1]s", "%*s", "%.2f", "%5%", "%q", "%!"} {
		if _, ok := literalFormat(format); ok {
			t.Fatalf("accepted nonliteral format %q", format)
		}
	}
}
