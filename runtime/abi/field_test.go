package abi

import (
	"strings"
	"testing"
)

func TestStructFieldNameEncoding(t *testing.T) {
	for _, n := range []int{1, 30, 31, 127, 128, 321} {
		name := strings.Repeat("A", n)
		for _, tag := range []string{"", "short", strings.Repeat("\x00", 129)} {
			f := NewStructField(name, nil, 17, tag, true)
			if f.Name() != name || f.Tag() != tag || !f.Embedded() || !f.Exported() || f.Offset != 17 {
				t.Fatalf("field encoding did not round trip (name length %d, tag length %d)", n, len(tag))
			}
		}
	}
	f := NewStructField("private", nil, 0, "", false)
	if f.Exported() || f.Embedded() || f.Blank() {
		t.Fatal("unexpected flags for a private, non-embedded field")
	}
	blank := NewStructField("_", nil, 0, "tag", false)
	if !blank.Blank() || blank.Tag() != "tag" {
		t.Fatal("blank field with tag did not round trip")
	}
}
