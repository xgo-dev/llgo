package binaryspecialize

import (
	"go/ast"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
			info := initial.TypesInfo
			initial.TypesInfo = nil
			if changed, err := Rewrite(closure, binary); len(changed) != 0 || err != nil {
				t.Fatalf("missing types produced %d rewritten packages, %v", len(changed), err)
			}
			initial.TypesInfo = info
			changed, err := Rewrite(closure, binary)
			if err != nil {
				t.Fatal(err)
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
type userOrder struct{ binary.ByteOrder }


func goodEncode() { binary.Encode(buf, binary.BigEndian, &value) }
func goodDecode() { binary.Decode(buf, binary.LittleEndian, new(record)) }
func goodPrivate() { binary.Encode(buf, binary.NativeEndian, private{}) }
func goodNilSize() { binary.Size(ptr) }
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
	if !shadowsGeneratedBuiltins(initial.Types) {
		t.Fatal("package-level len must suppress generated codecs")
	}
	for _, decl := range initial.Syntax[0].Decls {
		f, ok := decl.(*ast.FuncDecl)
		if !ok || len(f.Body.List) == 0 {
			continue
		}
		call := f.Body.List[0].(*ast.ExprStmt).X.(*ast.CallExpr)
		_, got := planCall(initial, call, calleeIdent(call.Fun).Name, initial.Imports["encoding/binary"].Types, math.MaxInt32)
		if want := strings.HasPrefix(f.Name.Name, "good"); got != want {
			t.Errorf("%s: eligible = %v, want %v", f.Name, got, want)
		}
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
