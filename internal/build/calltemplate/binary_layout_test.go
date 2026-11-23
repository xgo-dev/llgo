package calltemplate

import (
	"encoding/binary"
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func TestFixedLayout(t *testing.T) {
	rule, gen := binaryLayoutTemplate(t)
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
	wire := layoutForTest(t, rule, gen, typ)
	if wire == nil || wire["type"] != typ || wire["width"] != int64(binary.Size(record{})) {
		t.Fatalf("wire layout = %+v, want width %d", wire, binary.Size(record{}))
	}
	fields := wire["fields"].(map[string]any)
	for index, offset := range []int64{0, 1, 7} {
		field := fields[fmt.Sprintf("%08d", index)].(map[string]any)
		if got := field["offset"]; got != offset {
			t.Fatalf("field %d starts at %d, want %d", index, got, offset)
		}
	}
}

func TestFixedLayoutRejectsNonWireTypes(t *testing.T) {
	rule, gen := binaryLayoutTemplate(t)
	for _, text := range []string{"int", "uint", "uintptr", "string", "*uint16", "[]uint16",
		"struct{ _ int }", "[0]string", "map[int]uint32", "[268435456]uint64"} {
		t.Run(text, func(t *testing.T) {
			if wire := layoutForTest(t, rule, gen, checkedType(t, text)); wire != nil {
				t.Fatalf("unexpected wire layout %+v", wire)
			}
		})
	}
	for _, text := range []string{"[0]uint64", "struct{}", "[7]struct{}"} {
		if wire := layoutForTest(t, rule, gen, checkedType(t, text)); wire == nil || wire["width"] != int64(0) {
			t.Fatalf("%s: zero-width wire layout = %+v", text, wire)
		}
	}
}

func TestFixedLayoutLimitsGeneratedBody(t *testing.T) {
	rule, gen := binaryLayoutTemplate(t)
	var text strings.Builder
	text.WriteString("struct{")
	for i := range 256 {
		fmt.Fprintf(&text, " F%d uint8;", i)
	}
	text.WriteString(" }")
	if wire := layoutForTest(t, rule, gen, checkedType(t, text.String())); wire != nil {
		t.Fatalf("generated codec exceeded its node budget")
	}
}

func binaryLayoutTemplate(t *testing.T) (*template.Template, expansionContext) {
	t.Helper()
	initial := loadCodecInput(t, filepath.Join("..", "testdata", "binary-fixed"))
	initial.TypesSizes = types.SizesFor("gc", "386")
	library := initial.Imports["encoding/binary"]
	addCodecPatch(t, library, "templates_llgo.go")
	rule := templateRegistry(t, library).structural[0].entries["Size"].body
	gen := newExpansion(initial, templateDeps(initial), "test")
	return rule, expansionContext{g: gen, rule: rule}
}

func layoutForTest(t *testing.T, rule *template.Template, ctx expansionContext, typ types.Type) map[string]any {
	t.Helper()
	out := make(map[string]any)
	err := rule.ExecuteTemplate(new(strings.Builder), "layout", []any{ctx, typ, out, map[string]any{"remaining": int64(256)}, "size"})
	if errors.Is(err, errExpansionRejected) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func checkedType(t *testing.T, text string) types.Type {
	t.Helper()
	value, err := types.Eval(token.NewFileSet(), nil, token.NoPos, text)
	if err != nil {
		t.Fatal(err)
	}
	return value.Type
}
