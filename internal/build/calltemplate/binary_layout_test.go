package calltemplate

import (
	"encoding/binary"
	"fmt"
	"go/token"
	"go/types"
	"math"
	"strings"
	"testing"
)

func TestFixedLayout(t *testing.T) {
	type word uint16
	type record struct {
		Flag bool
		_    [3]word
		Data [2]struct {
			Start int8
			Value complex128
		}
	}
	goType := "struct{ Flag bool; _ [3]uint16; Data [2]struct{ Start int8; Value complex128 } }"
	typ := checkedType(t, goType)
	wire := fixedLayout(typ, math.MaxInt32)
	if wire == nil || wire.typ != typ || wire.width != int64(binary.Size(record{})) {
		t.Fatalf("wire layout = %+v, want width %d", wire, binary.Size(record{}))
	}
	for index, offset := range []int64{0, 1, 7} {
		if got := wire.fields[index].offset; got != offset {
			t.Fatalf("field %d starts at %d, want %d", index, got, offset)
		}
	}
	if got := wire.fields[1].field.Name(); got != "_" {
		t.Fatalf("blank field recorded as %q", got)
	}
	if got := wire.fields[2].value.elem.fields[1].offset; got != 1 {
		t.Fatalf("complex value alignment introduced padding at %d", got)
	}
}

func TestFixedLayoutRejectsNonWireTypes(t *testing.T) {
	for _, text := range []string{"int", "uint", "uintptr", "string", "*uint16", "[]uint16",
		"struct{ _ int }", "[0]string", "map[int]uint32", "interface{}", "[268435456]uint64"} {
		t.Run(text, func(t *testing.T) {
			if wire := fixedLayout(checkedType(t, text), math.MaxInt32); wire != nil {
				t.Fatalf("unexpected wire layout %+v", wire)
			}
		})
	}
	// A zero-width array of fixed-width elements and an empty record are valid.
	for _, text := range []string{"[0]uint64", "struct{}", "[7]struct{}"} {
		wire := fixedLayout(checkedType(t, text), math.MaxInt32)
		if wire == nil || wire.width != 0 {
			t.Fatalf("%s: zero-width wire layout = %+v", text, wire)
		}
	}
}

func TestFixedLayoutChecksWidthLimit(t *testing.T) {
	for _, text := range []string{"uint64", "struct{ A uint32; B uint32 }", "[2]uint32"} {
		if wire := fixedLayout(checkedType(t, text), 4); wire != nil {
			t.Fatalf("%s exceeds the target width limit: %+v", text, wire)
		}
	}
}

func TestFixedLayoutLimitsGeneratedBody(t *testing.T) {
	var text strings.Builder
	text.WriteString("struct{")
	for i := 0; i < maxCodecNodes; i++ {
		fmt.Fprintf(&text, " F%d uint8;", i)
	}
	text.WriteString(" }")
	if wire := fixedLayout(checkedType(t, text.String()), math.MaxInt32); wire != nil {
		t.Fatalf("generated codec exceeded its node budget")
	}
}

func checkedType(t *testing.T, text string) types.Type {
	t.Helper()
	value, err := types.Eval(token.NewFileSet(), nil, token.NoPos, text)
	if err != nil {
		t.Fatal(err)
	}
	return value.Type
}
