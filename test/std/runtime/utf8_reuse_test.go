package runtime_test

import (
	"bytes"
	"testing"
	"unicode/utf8"
)

func TestRuntimeUTF8MatchesUnicode(t *testing.T) {
	// Every scalar value, invalid scalar values, and encoding boundaries pass
	// through runtime.encoderune via string/rune-slice conversions.
	for r := rune(-2); r <= utf8.MaxRune+2; r++ {
		want := utf8.AppendRune(nil, r)
		if got := []byte(string(r)); !bytes.Equal(got, want) {
			t.Fatalf("string(%U) = %x, want %x", r, got, want)
		}
	}
	inputs := []string{
		"", "ascii", "a中🙂z", "\x80", "\xc0\x80", "\xe0\x80\x80",
		"\xed\xa0\x80", "\xf0\x80\x80\x80", "\xf4\x90\x80\x80",
		"\xf5\x80\x80\x80", "\xe2\x82", "\xf0\x9f\x99", "\xff\xfe",
	}
	// Exhaust the two-byte input space, including truncated/overlong encodings.
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			inputs = append(inputs, string([]byte{byte(a), byte(b)}))
		}
	}
	for _, s := range inputs {
		var want []rune
		offset := 0
		for i, got := range s {
			r, size := utf8.DecodeRuneInString(s[offset:])
			if i != offset || got != r {
				t.Fatalf("range %x at %d: (%d, %U), want (%d, %U)", s, offset, i, got, offset, r)
			}
			want = append(want, r)
			offset += size
		}
		got := []rune(s)
		if offset != len(s) || len(got) != len(want) {
			t.Fatalf("conversion length for %x: %d, want %d", s, len(got), len(want))
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("conversion of %x at %d: %U, want %U", s, i, got[i], want[i])
			}
		}
		if string(got) != string(want) {
			t.Fatalf("rune slice encoding of %x differs", s)
		}
	}
}
