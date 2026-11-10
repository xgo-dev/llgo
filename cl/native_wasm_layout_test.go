package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	gpackages "github.com/goplus/gogen/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestWasmNativeRecordLayoutConstants(t *testing.T) {
	const src = `package native_layout
import "unsafe"
//DIRECTIVE
type Record struct { Data unsafe.Pointer; Size uintptr; Tag int32 }
//DIRECTIVE
type Outer struct { R Record; Pointers [2]*byte; Words [2]uintptr; Tail [0]uint64 }
//DIRECTIVE
type Words [2]uintptr
type GoRecord struct { Data unsafe.Pointer; Size uintptr; Tag int32 }
type Wrapper struct { B byte; R Record; P *byte }
var _ [12-int(unsafe.Sizeof(Record{}))]byte
var _ [int(unsafe.Sizeof(Record{}))-12]byte
var _ [4-int(unsafe.Alignof(Record{}))]byte
var _ [int(unsafe.Alignof(Record{}))-4]byte
var _ [4-int(unsafe.Offsetof(Record{}.Size))]byte
var _ [int(unsafe.Offsetof(Record{}.Size))-4]byte
var _ [24-int(unsafe.Sizeof([2]Record{}))]byte
var _ [int(unsafe.Sizeof([2]Record{}))-24]byte
var _ [8-int(unsafe.Sizeof(Words{}))]byte
var _ [int(unsafe.Sizeof(Words{}))-8]byte
var _ [32-int(unsafe.Sizeof(Outer{}))]byte
var _ [int(unsafe.Sizeof(Outer{}))-32]byte
var _ [24-int(unsafe.Sizeof(GoRecord{}))]byte
var _ [int(unsafe.Sizeof(GoRecord{}))-24]byte
var _ [24-int(unsafe.Sizeof(Wrapper{}))]byte
var _ [int(unsafe.Sizeof(Wrapper{}))-24]byte
var _ [16-int(unsafe.Offsetof(Wrapper{}.P))]byte
var _ [int(unsafe.Offsetof(Wrapper{}.P))-16]byte
func Size() uintptr { return unsafe.Sizeof(Record{}) }
func Next(records *[2]Record) *Record { return &records[1] }
func SetSize(r *Record, n uintptr) { r.Size = n }
func SetWord(w *Words, n uintptr) { w[1] = n }
`
	for _, directive := range []string{"//llgo:cdecl", "//llgo:type C"} {
		for _, target := range []*llssa.Target{
			{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32"},
			{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"},
		} {
			t.Run(directive+"/"+target.GOOS, func(t *testing.T) {
				fset := token.NewFileSet()
				file, err := parser.ParseFile(fset, "native.go", strings.ReplaceAll(src, "//DIRECTIVE", directive), parser.ParseComments)
				if err != nil {
					t.Fatal(err)
				}
				imp := gpackages.NewImporter(fset)
				prog := ssatest.NewProgramEx(t, target, imp)
				raw := types.NewPackage("native_layout", "native_layout")
				files := []*ast.File{file}
				// Production loads directives before checking unsafe constants.
				if err := ParsePkgSyntax(prog, fset, raw, files); err != nil {
					t.Fatal(err)
				}
				pkg, _, err := ssautil.BuildPackage(&types.Config{Importer: imp, Sizes: prog.TypeSizes(types.SizesFor("gc", "wasm"))}, fset, raw, files, ssa.SanityCheckFunctions)
				if err != nil {
					t.Fatal(err)
				}
				compiled, _, err := NewPackageEx(prog, nil, nil, pkg, files)
				if err != nil {
					t.Fatal(err)
				}
				ir := compiled.String()
				if !strings.Contains(ir, "ret i64 12") {
					t.Fatalf("native size constant disagrees with storage:\n%s", ir)
				}
			})
		}
	}
}
