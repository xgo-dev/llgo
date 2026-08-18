//go:build go1.24

package bytes_test

import (
	"bytes"
	"iter"
	"testing"
)

func collectByteSeq(seq iter.Seq[[]byte]) [][]byte {
	var out [][]byte
	for chunk := range seq {
		out = append(out, append([]byte(nil), chunk...))
	}
	return out
}

func TestBytesSequenceIterators(t *testing.T) {
	fields := collectByteSeq(bytes.FieldsSeq([]byte("  a  b c\t")))
	if want := [][]byte{[]byte("a"), []byte("b"), []byte("c")}; !equalByteSlices(fields, want) {
		t.Fatalf("FieldsSeq mismatch: %v", fields)
	}

	fieldsFunc := collectByteSeq(bytes.FieldsFuncSeq([]byte("a|b||c"), func(r rune) bool { return r == '|' }))
	if want := [][]byte{[]byte("a"), []byte("b"), []byte("c")}; !equalByteSlices(fieldsFunc, want) {
		t.Fatalf("FieldsFuncSeq mismatch: %v", fieldsFunc)
	}

	lines := collectByteSeq(bytes.Lines([]byte("a\nb\n")))
	if want := [][]byte{[]byte("a\n"), []byte("b\n")}; !equalByteSlices(lines, want) {
		t.Fatalf("Lines mismatch: %v", lines)
	}

	linesSingle := collectByteSeq(bytes.Lines([]byte("single")))
	if want := [][]byte{[]byte("single")}; !equalByteSlices(linesSingle, want) {
		t.Fatalf("Lines single mismatch: %v", linesSingle)
	}

	splitSeq := collectByteSeq(bytes.SplitSeq([]byte("a,b,c"), []byte(",")))
	if want := [][]byte{[]byte("a"), []byte("b"), []byte("c")}; !equalByteSlices(splitSeq, want) {
		t.Fatalf("SplitSeq mismatch: %v", splitSeq)
	}

	splitAfterSeq := collectByteSeq(bytes.SplitAfterSeq([]byte("a,b,c"), []byte(",")))
	if want := [][]byte{[]byte("a,"), []byte("b,"), []byte("c")}; !equalByteSlices(splitAfterSeq, want) {
		t.Fatalf("SplitAfterSeq mismatch: %v", splitAfterSeq)
	}
}
