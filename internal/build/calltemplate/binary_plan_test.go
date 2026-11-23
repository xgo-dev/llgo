package calltemplate

import (
	"errors"
	"go/ast"
	"go/types"
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
			addCodecPatch(t, binary, "templates_llgo.go", "templates_go123.go")
			registry := templateRegistry(t, binary)
			binding := registry.structural[0]
			info := initial.TypesInfo
			initial.TypesInfo = nil
			if used, err := registry.RewriteStructural([]*packages.Package{initial}, nil); used.Providers[binary.PkgPath] || err != nil {
				t.Fatalf("missing types rewrote calls = %v, %v", used, err)
			}
			initial.TypesInfo = info
			used, err := registry.RewriteStructural([]*packages.Package{initial}, nil)
			changed := used.Providers[binary.PkgPath]
			if err != nil {
				t.Fatal(err)
			}
			if changed != fixture.want {
				t.Fatalf("rewrite = %v, want %v", changed, fixture.want)
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
						if callee.Pkg != nil && callee.Pkg.Pkg == binary.Types && binding.hasEntry(callee.Name()) {
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

func addCodecPatch(t *testing.T, library *packages.Package, files ...string) {
	t.Helper()
	for _, file := range files {
		path := filepath.Join("..", "..", "..", "runtime", "_patch", library.PkgPath, file)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := typesynth.AddFile(library, file, string(data), nil, ""); err != nil {
			t.Fatal(err)
		}
	}
}

func templateRegistry(t *testing.T, library *packages.Package) *Bindings {
	t.Helper()
	registry, err := LoadBindings([]*packages.Package{library}, func(p *packages.Package) bool { return p == library })
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.structural) != 1 {
		t.Fatalf("got %d declared codecs", len(registry.structural))
	}
	return registry
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
	library := initial.Imports["encoding/binary"]
	addCodecPatch(t, library, "templates_llgo.go", "templates_go123.go")
	registry := templateRegistry(t, library)
	binding := registry.structural[0]
	deps := templateDeps(initial)
	for _, decl := range initial.Syntax[0].Decls {
		f, ok := decl.(*ast.FuncDecl)
		if !ok || len(f.Body.List) == 0 {
			continue
		}
		call := f.Body.List[0].(*ast.ExprStmt).X.(*ast.CallExpr)
		fn, _ := typesynth.Callee(initial.TypesInfo, call.Fun)
		rule := binding.entries[fn.Name()]
		gen := newExpansion(initial, deps, "test")
		var eligible bool
		if rule.helper != nil {
			match, inst := selectCandidate(initial.TypesInfo, call, fn.Signature(), []candidate{{fn: rule.helper}})
			if match != nil {
				_, err := gen.expand(rule, inst.Type.(*types.Signature), call)
				if err != nil && !errors.Is(err, errExpansionRejected) {
					t.Fatal(err)
				}
				eligible = err == nil
			}
		}
		if want := strings.HasPrefix(f.Name.Name, "good"); eligible != want {
			t.Errorf("%s: eligible = %v, want %v", f.Name, eligible, want)
		}
		if eligible && fn.Name() != "Size" {
			unchecked := !strings.Contains(gen.functions.String(), "LLGoBinaryShortBuffer")
			if want := f.Name.Name == "goodFixedBounds"; unchecked != want {
				t.Errorf("%s: unchecked = %v, want %v", f.Name, unchecked, want)
			}
		}
	}
	if used, err := registry.RewriteStructural([]*packages.Package{initial}, nil); used.Providers[library.PkgPath] || err != nil {
		t.Fatalf("mixed generic calls rewrote binary: %v, %v", used, err)
	}

}

func templateDeps(initial *packages.Package) map[string]*packages.Package {
	deps := make(map[string]*packages.Package)
	packages.Visit([]*packages.Package{initial}, func(p *packages.Package) bool {
		deps[p.PkgPath] = p
		return true
	}, nil)
	return deps
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
