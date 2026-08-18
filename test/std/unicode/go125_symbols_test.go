//go:build go1.25

package unicode_test

import (
	"testing"
	"unicode"
)

func TestGo126Categories(t *testing.T) {
	if got := unicode.CategoryAliases["Unassigned"]; got != "Cn" {
		t.Fatalf("CategoryAliases[Unassigned] = %q, want Cn", got)
	}
	if got := unicode.CategoryAliases["Cased_Letter"]; got != "LC" {
		t.Fatalf("CategoryAliases[Cased_Letter] = %q, want LC", got)
	}
	if !unicode.Is(unicode.Cn, '\u0378') || unicode.Is(unicode.Cn, 'A') {
		t.Fatal("Cn does not identify unassigned code points")
	}
	if !unicode.Is(unicode.LC, 'A') || !unicode.Is(unicode.LC, 'a') || unicode.Is(unicode.LC, '1') {
		t.Fatal("LC does not identify cased letters")
	}
}
