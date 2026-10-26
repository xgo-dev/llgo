package midway

import (
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/directive"
)

func TestRewriteInferredAggregateTypes(t *testing.T) {
	imports := testSIMDImports(t)
	pkg, _ := testCheck(t, imports, "aggregate", `package aggregate
import ("simd"; "unsafe")
type Box[T any] struct { Value T }
type Alias[T any] = Box[T]
type Local interface { Value() simd.Uint32s }
func makeAggregate() struct {
 simd.Uint32s
 Ptr *simd.Uint32s
 Slice []simd.Uint32s
 Array [2]simd.Uint32s
 Map map[int]simd.Uint32s
 Channel chan simd.Uint32s
 Receive <-chan simd.Uint32s
 Send chan<- simd.Uint32s
 Function func(int, ...simd.Uint32s) (simd.Uint32s, error)
 Interface interface { Local; More() simd.Uint32s }
 Generic Box[simd.Uint32s]
 Alias Alias[simd.Uint32s]
 Pointer unsafe.Pointer
 Tagged simd.Uint32s `+"`json:\"value\"`"+`
} { panic("fixture") }
var Value = makeAggregate()
`, true)
	for _, width := range widths {
		value := pkg.Scope().Lookup("Value" + suffix(width))
		if value == nil {
			t.Fatalf("missing aggregate variant %d", width)
		}
		fields := value.Type().(*types.Struct)
		if tag := fields.Tag(fields.NumFields() - 1); tag != `json:"value"` {
			t.Errorf("lost struct tag: %q", tag)
		}
	}
}

func TestRewriteRetainsFirstTypeError(t *testing.T) {
	for _, tc := range []struct{ name, declarations, result, want string }{
		{"fields", "", "struct { first simd.Uint32s; last simd.Uint32s }", "field first"},
		{"nested fields", "", "struct { Nested struct { first simd.Uint32s }; last simd.Uint32s }", "field first"},
		{"methods", "", "interface { first() simd.Uint32s; last() simd.Uint32s }", "method first"},
		{"nested methods", "type hidden struct { V simd.Uint32s }", "interface { First() hidden; last() simd.Uint32s }", "type hidden"},
		{"named type", "type hidden struct { V simd.Uint32s }", "hidden", "type hidden"},
		{"alias", "type hidden = struct { V simd.Uint32s }", "hidden", "type hidden"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			imports := testImporter{}
			testCheck(t, imports, simdPath, "package simd\ntype Uint32s struct{}\n", false)
			testCheck(t, imports, "foreign", "package foreign\nimport \"simd\"\n"+tc.declarations+"\nfunc Make() "+tc.result+" { panic(0) }", false)
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "client.go", `package client
import "foreign"
var Value = foreign.Make()
func Later() { _ = foreign.Make() }
`, 0)
			if err != nil {
				t.Fatal(err)
			}
			info := testInfo()
			cfg := types.Config{Importer: imports}
			pkg, err := cfg.Check("client", fset, []*ast.File{file}, info)
			if err != nil {
				t.Fatal(err)
			}
			later := file.Decls[2].(*ast.FuncDecl)
			body := later.Body
			changed, _, err := Rewrite(fset, pkg, info, []*ast.File{file}, new(directive.Index))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want first unnameable %s", err, tc.want)
			}
			if changed || later.Body != body {
				t.Fatal("rewrite continued past the failed declaration")
			}
		})
	}
}

func TestRewriteImportShadowingAcrossFiles(t *testing.T) {
	imports := testSIMDImports(t)
	fset := token.NewFileSet()
	var files []*ast.File
	for name, src := range map[string]string{
		"shadow.go":    `package client; import s "simd"; type V = s.Uint32s; func Shadow(s int) int { var v V; return s+v.Len() }`,
		"collision.go": `package client; var llgoMidway1 = 1; var llgoMidway1_ = 2`,
		"plain.go":     `package client; import s "simd"; func Plain() int { var v s.Uint32s; return v.Len() }`,
	} {
		file, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, file)
	}
	info := testInfo()
	cfg := types.Config{Importer: imports}
	pkg, err := cfg.Check("client", fset, files, info)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Rewrite(fset, pkg, info, files, new(directive.Index)); err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.Check("client", fset, files, testInfo()); err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "Plain" && fn.Name.Name != "Shadow" {
				continue
			}
			call := fn.Body.List[0].(*ast.IfStmt).Cond.(*ast.CallExpr)
			alias := call.Fun.(*ast.SelectorExpr).X.(*ast.Ident).Name
			if (alias == "s") != (fn.Name.Name == "Plain") {
				t.Errorf("%s uses import alias %s", fn.Name.Name, alias)
			}
		}
	}
}

func TestRewriteTypeIdentityAndLayout(t *testing.T) {
	// Distinct layouts prove that constants and array lengths are rechecked.
	imports := testSIMDImports(t)
	pkg, _ := testCheck(t, imports, "lib", `package lib
import "simd"
import "unsafe"
type Node struct { Next *Node; Peer *Peer; Value simd.Uint32s }
type Peer struct { Next *Node }
type Embedded struct { simd.Uint32s }
type Alias = simd.Uint32s
type ConstrainedAlias[T interface { simd.Uint32s }] = int
func AliasConstraint() ConstrainedAlias[simd.Uint32s] { return 1 }
type EmbeddedAlias struct { Alias }
func NewAlias() EmbeddedAlias { return EmbeddedAlias{Alias: simd.BroadcastUint32s(1)} }
func UseAlias(x EmbeddedAlias) int { return x.Alias.Len() }
func ShadowPanic(panic simd.Uint32s) simd.Uint32s { return panic }
type Box[T any] struct { Value T }
const ( Size = unsafe.Sizeof(simd.Uint32s{}) + iota; Next )
type Bytes [Size]byte
func New() Embedded { return Embedded{Uint32s: simd.BroadcastUint32s(1)} }
func Use(x Embedded) int { return x.Uint32s.Len() }
func Generic[T interface{ simd.Uint32s }](x T) T { return x }
func Recursive(x *Peer) *Peer { if x.Next != nil { return Recursive(x.Next.Peer) }; return x }
func Scalar[T ~int](_ T, xs ...T) int { var v simd.Uint32s; return v.Len()+len(xs) }
func Shadow(simd int) int { var v Alias; return v.Len()+simd }
func Dot() Box[simd.Uint32s] { return Box[simd.Uint32s]{} }
`, true)
	for _, name := range []string{"Node", "Peer", "Embedded", "Alias", "Bytes", "New", "Use", "Generic", "Recursive", "Scalar", "Shadow", "Dot"} {
		for _, width := range widths {
			if pkg.Scope().Lookup(name+suffix(width)) == nil {
				t.Errorf("missing %s%s", name, suffix(width))
			}
		}
	}
	for _, tc := range []struct {
		name  string
		value int64
	}{{"Size@simd0", 32}, {"Size@simd128", 16}, {"Next@simd0", 33}, {"Next@simd128", 17}} {
		obj := pkg.Scope().Lookup(tc.name).(*types.Const)
		if got, _ := constant.Int64Val(obj.Val()); got != tc.value {
			t.Errorf("%s = %d, want %d", tc.name, got, tc.value)
		}
	}
	testCheck(t, imports, "client", `package client
import "lib"
func RoundTrip() int { x:=lib.New(); return lib.Use(x)+x.Uint32s.Len() }
func Box() int { return lib.Dot().Value.Len() }
`, true)
}
