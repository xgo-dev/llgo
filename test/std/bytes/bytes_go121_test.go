//go:build go1.21

package bytes_test

import (
	"bytes"
	"testing"
	"unicode"
)

func TestBytesGo121Functions(t *testing.T) {
	if !bytes.ContainsFunc([]byte("abc123"), unicode.IsDigit) {
		t.Fatal("ContainsFunc should locate digit rune")
	}

	var buf bytes.Buffer
	buf.Grow(16)
	initialAvail := buf.Available()
	if initialAvail <= 0 {
		t.Fatalf("Available should report spare capacity, got %d", initialAvail)
	}

	space := buf.AvailableBuffer()
	if len(space) != 0 || cap(space) != initialAvail {
		t.Fatalf("AvailableBuffer mismatch: len=%d cap=%d want cap=%d", len(space), cap(space), initialAvail)
	}
	space = append(space, 'G', 'o')
	if n, err := buf.Write(space); err != nil || n != len(space) {
		t.Fatalf("Write via AvailableBuffer mismatch: n=%d err=%v", n, err)
	}
	if buf.Available() != initialAvail-len(space) {
		t.Fatalf("Available after write mismatch: %d", buf.Available())
	}
}
