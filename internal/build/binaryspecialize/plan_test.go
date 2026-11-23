package binaryspecialize

import (
	"go/ast"
	"go/types"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/build/typesynth"
	"github.com/xgo-dev/llgo/internal/packages"
	gopackages "golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestRewriteBinaryCodecs(t *testing.T) {
	for _, fixture := range []struct {
		name string
		want bool
	}{
		{"binary-fixed", true},
		{"binary-mixed", false},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			initial := loadCodecInput(t, filepath.Join("..", "testdata", fixture.name))
			binary := initial.Imports["encoding/binary"]
			var closure []*packages.Package
			packages.Visit([]*packages.Package{initial}, nil, func(p *packages.Package) {
				closure = append(closure, p)
			})
			shortBuffer, err := AddShortBufferAccessor(binary)
			if err != nil {
				t.Fatal(err)
			}
			info := initial.TypesInfo
			initial.TypesInfo = nil
			if changed, err := Rewrite(closure, binary, initial, shortBuffer); len(changed) != 0 || err != nil {
				t.Fatalf("missing types produced %d rewritten packages, %v", len(changed), err)
			}
			initial.TypesInfo = info
			if changed, err := Rewrite(closure, binary, nil, shortBuffer); len(changed) != 0 || err != nil {
				t.Fatalf("dependency produced %d rewritten packages, %v", len(changed), err)
			}
			if changed, err := Rewrite(closure, binary, initial, ""); len(changed) != 0 || err != nil {
				t.Fatalf("missing short buffer error produced %d rewritten packages, %v", len(changed), err)
			}
			if fixture.want {
				// Floating-point wire conversion requires math from the same
				// standard library. A missing dependency must be reported.
				dep := binary.Imports["math"]
				delete(binary.Imports, "math")
				if _, err := Rewrite(closure, binary, initial, shortBuffer); err == nil {
					t.Fatal("generated binary codec accepted a missing math dependency")
				}
				binary.Imports["math"] = dep
			}
			changed, err := Rewrite(closure, binary, initial, shortBuffer)
			if err != nil {
				t.Fatal(err)
			}
			if len(changed) > 1 || len(changed) == 1 && changed[0] != initial {
				t.Fatal("only the executable package can be rewritten")
			}
			if got := len(changed) != 0; got != fixture.want {
				t.Fatalf("rewrite = %v, want %v", got, fixture.want)
			}

			prog, _ := ssautil.AllPackages([]*packages.Package{initial}, ssa.SanityCheckFunctions|ssa.InstantiateGenerics)
			prog.Build()
			var generic, specialized int
			for f := range ssautil.AllFunctions(prog) {
				if f.Pkg == nil || f.Pkg.Pkg != initial.Types {
					continue
				}
				for _, b := range f.Blocks {
					for _, inst := range b.Instrs {
						call, ok := inst.(*ssa.Call)
						if !ok || call.Common().StaticCallee() == nil {
							continue
						}
						callee := call.Common().StaticCallee()
						if callee.Pkg != nil && callee.Pkg.Pkg == binary.Types && genericOperation(callee.Name()) {
							generic++
						}
						if callee.Pkg == f.Pkg && strings.HasPrefix(callee.Name(), "__llgo_binary_") {
							specialized++
							for _, arg := range call.Common().Args {
								if _, boxed := arg.(*ssa.MakeInterface); boxed {
									t.Error("typed codec retained interface boxing")
								}
							}
						}
					}
				}
			}
			if fixture.want && (generic != 0 || specialized == 0) {
				t.Fatalf("got %d generic and %d typed calls", generic, specialized)
			}
			if !fixture.want && (generic == 0 || specialized != 0) {
				t.Fatalf("mixed input: got %d generic and %d typed calls", generic, specialized)
			}
		})
	}
}

func TestShortBufferAccessorContract(t *testing.T) {
	pkg := &packages.Package{Types: types.NewPackage("encoding/binary", "binary")}
	if name, err := AddShortBufferAccessor(pkg); name != "" || err != nil {
		t.Fatalf("missing short buffer sentinel returned %q, %v", name, err)
	}
	pkg.Types.Scope().Insert(types.NewVar(0, pkg.Types, "errBufferTooSmall", types.Typ[types.String]))
	if name, err := AddShortBufferAccessor(pkg); name != "" || err != nil {
		t.Fatalf("incompatible short buffer sentinel returned %q, %v", name, err)
	}

	initial := loadCodecInput(t, filepath.Join("..", "testdata", "binary-fixed"))
	binary := initial.Imports["encoding/binary"]
	base, err := AddShortBufferAccessor(binary)
	if err != nil || base == "" {
		t.Fatalf("first short buffer accessor = %q, %v", base, err)
	}
	next, err := AddShortBufferAccessor(binary)
	if err != nil || next == "" || next == base {
		t.Fatalf("second short buffer accessor = %q, %v", next, err)
	}
}

func TestBinaryCallEligibility(t *testing.T) {
	dir := t.TempDir()
	source := `package main
import "encoding/binary"

type record struct{ N uint16 }
type private struct{ n uint16 }
var buf []byte
var value record
var ptr *record
var order binary.ByteOrder
var boxed any = value
var len int
var array [8]byte
var pointer *[8]byte
var low int
type userOrder struct{ binary.ByteOrder }


func goodEncode() { binary.Encode(buf, binary.BigEndian, &value) }
func goodDecode() { binary.Decode(buf, binary.LittleEndian, new(record)) }
func goodPrivate() { binary.Encode(buf, binary.NativeEndian, private{}) }
func goodNilSize() { binary.Size(ptr) }
func goodFixedBounds() { binary.Encode(pointer[1:7], binary.BigEndian, value) }
func goodDynamicBounds() { binary.Encode(array[low:], binary.BigEndian, value) }
func goodSliceBounds() { binary.Encode(buf[1:7], binary.BigEndian, value) }
func noOrder() { binary.Encode(buf, order, value) }
func noBoxed() { binary.Encode(buf, binary.BigEndian, boxed) }
func noLocalOrder() { binary.Encode(buf, userOrder{binary.BigEndian}, value) }
func noNilDecode() { binary.Decode(buf, binary.BigEndian, ptr) }
func noValueDecode() { binary.Decode(buf, binary.BigEndian, value) }
func noPrivateDecode() { binary.Decode(buf, binary.BigEndian, new(private)) }
func noBasic() { binary.Encode(buf, binary.BigEndian, uint16(0)) }
func noMap() { binary.Size(map[int]record(nil)) }
func noIntField() { binary.Size(struct{ N int }{}) }
func noAppend() { binary.Append(buf, binary.BigEndian, value) }
func main() {}
`
	for name, contents := range map[string]string{"go.mod": "module example.com/codecs\ngo 1.23\n", "main.go": source} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	initial := loadCodecInput(t, dir)
	initial.TypesSizes = types.SizesFor("gc", "386")
	if !shadowsGeneratedBuiltins(initial.Types) {
		t.Fatal("package-level len must suppress generated codecs")
	}
	for _, decl := range initial.Syntax[0].Decls {
		f, ok := decl.(*ast.FuncDecl)
		if !ok || len(f.Body.List) == 0 {
			continue
		}
		call := f.Body.List[0].(*ast.ExprStmt).X.(*ast.CallExpr)
		fn, _ := typesynth.Callee(initial.TypesInfo, call.Fun)
		plan, got := planCall(initial, call, fn.Name(), initial.Imports["encoding/binary"].Types, math.MaxInt32)
		if want := strings.HasPrefix(f.Name.Name, "good"); got != want {
			t.Errorf("%s: eligible = %v, want %v", f.Name, got, want)
		}
		if want := f.Name.Name == "goodFixedBounds"; plan.unchecked != want {
			t.Errorf("%s: unchecked = %v, want %v", f.Name, plan.unchecked, want)
		}
	}
	if _, ok := planPackage(initial, initial.Imports["encoding/binary"].Types); ok {
		t.Fatal("mixed generic calls or shadowed builtins enabled binary specialization")
	}
}

func loadCodecInput(t *testing.T, dir string) *packages.Package {
	t.Helper()
	cfg := gopackages.Config{Mode: gopackages.LoadAllSyntax, Dir: dir}
	initial, err := gopackages.Load(&cfg, ".")
	if err != nil || len(initial) != 1 {
		t.Fatalf("load %s: %v", dir, err)
	}
	if gopackages.PrintErrors(initial) != 0 {
		t.Fatalf("load %s: type errors", dir)
	}
	return initial[0]
}
