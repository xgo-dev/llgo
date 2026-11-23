package compile

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/types/typeutil"
)

func TestMarshalShapeEligibility(t *testing.T) {
	const source = `package example
	import "encoding/json"
	type Marshaler string
	func (*Marshaler) MarshalText() ([]byte, error) { return nil, nil }
	type Graph map[string]Graph
	type Embedded struct{ Value int }
	var basic int32
	var named bool
	var record struct { Key []int; Value map[string]bool }
	var array [3]uint8
	var pointer *struct{ Value uint }
	var text Marshaler
	var graph Graph
	var bytes []byte
	var duplicate struct { X int ~json:"same"~; Y int ~json:"same"~ }
	var tagged struct { Value int ~json:",string"~ }
	var embedded struct { Embedded }
	var floating float64
	var dynamic any
	var wrongKey map[int]int
	var standardNumber json.Number
	var unsupportedField struct { Value any }
	var invalidTag struct { Value int ~json:"x\\y"~ }
	var omittedOptionalFields struct {
		Pointer *int ~json:",omitempty"~
		Flag bool ~json:",omitempty"~
		Text string ~json:",omitempty"~
		Number int ~json:",omitempty"~
		Object struct{} ~json:",omitempty"~
	}
	`
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "types.go", strings.ReplaceAll(source, "~", "`"), 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := (&types.Config{Importer: importer.Default()}).Check("example", files, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var known typeutil.Map
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"basic", true}, {"named", true}, {"record", true}, {"array", true}, {"pointer", true},
		{"text", false}, {"graph", false}, {"bytes", false}, {"duplicate", false},
		{"tagged", false}, {"embedded", false}, {"floating", false}, {"dynamic", false}, {"wrongKey", false},
		{"standardNumber", false}, {"unsupportedField", false}, {"invalidTag", false},
	} {
		if got := planShape(pkg.Scope().Lookup(tc.name).Type(), &known) != nil; got != tc.want {
			t.Errorf("%s: eligible %v, want %v", tc.name, got, tc.want)
		}
	}
}
