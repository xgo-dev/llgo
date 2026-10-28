package ccgobridge

import (
	"bytes"
	"context"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const exportHelper = `
func __ccgo_fp(f interface{}) uintptr {
	type iface [2]uintptr
	return (*iface)(unsafe.Pointer(&f))[1]
}
`

func rewriteSource(t *testing.T, source string) (string, Stats, error) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "fixture.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	pkg, err := (&types.Config{Importer: importer.Default()}).Check("fixture", fset, []*ast.File{f}, info)
	if err != nil {
		t.Fatal(err)
	}
	stats, rewriteErr := Rewrite(fset, pkg, info, []*ast.File{f})
	var out bytes.Buffer
	if err := format.Node(&out, fset, f); err != nil {
		t.Fatal(err)
	}
	if rewriteErr == nil {
		// Check fresh syntax, not stale types.Info from before the rewrite.
		checkSet := token.NewFileSet()
		checkFile, err := parser.ParseFile(checkSet, "rewritten.go", out.Bytes(), 0)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := (&types.Config{Importer: importer.Default()}).Check("fixture", checkSet, []*ast.File{checkFile}, nil); err != nil {
			t.Fatalf("rewritten source does not type-check: %v\n%s", err, out.String())
		}
	}
	return out.String(), stats, rewriteErr
}

func TestStaticBoxAndConsumer(t *testing.T) {
	out, stats, err := rewriteSource(t, `package fixture
import "unsafe"
`+exportHelper+`
func twice(n int) int { return n*2 }
var first = __ccgo_fp(twice)
var second = __ccgo_fp(twice)
func call(p uintptr) int {
	return (*(*func(int) int)(unsafe.Pointer(&struct{ uintptr }{p})))(21)
}
`)
	if err != nil {
		t.Fatal(err)
	}
	if stats.StaticBoxes != 1 || stats.Exports != 2 || stats.Imports != 1 {
		t.Fatalf("unexpected counts: %+v", stats)
	}
	if !strings.Contains(out, "var __llgo_ccgo_box_0 __llgo_ccgo_staticBox[func(n int) int]") ||
		strings.Count(out, "__llgo_ccgo_export(&__llgo_ccgo_box_0, twice)") != 2 ||
		!strings.Contains(out, "__llgo_ccgo_import[func(int) int](unsafe.Pointer(p))") {
		t.Fatalf("not a full typed box round trip:\n%s", out)
	}
}

func TestGenericStaticExporter(t *testing.T) {
	out, stats, err := rewriteSource(t, `package fixture
import "unsafe"
func cFuncPointer[T any](f T) uintptr {
	return *(*uintptr)(unsafe.Pointer(&struct{ f T }{f}))
}

func callback() {}
var p = cFuncPointer(callback)
`)
	if err != nil || stats.Exports != 1 || !strings.Contains(out, "__llgo_ccgo_export(&__llgo_ccgo_box_0, callback)") {
		t.Fatalf("generic exporter: %+v, %v\n%s", stats, err, out)
	}
}

func TestStaticBoxDoesNotIntroduceInitializationCycle(t *testing.T) {
	_, _, err := rewriteSource(t, `package fixture
import "unsafe"
`+exportHelper+`
func recursive() uintptr { return __ccgo_fp(recursive) }
`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExportedFunctionsUsesTypedDeclarations(t *testing.T) {
	const source = `package fixture
import (
	"unsafe"
	text "strings"
)
` + exportHelper + `
func local() {}
var first = __ccgo_fp(local)
var second = __ccgo_fp(local)
var imported = __ccgo_fp(text.TrimSpace)
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "exports.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	if _, err := (&types.Config{Importer: importer.Default()}).Check("fixture", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	var before, after bytes.Buffer
	if err := format.Node(&before, fset, file); err != nil {
		t.Fatal(err)
	}
	functions, err := ExportedFunctions(fset, info, []*ast.File{file})
	if err != nil {
		t.Fatal(err)
	}
	if len(functions) != 2 || functions[0].Pkg().Path() != "fixture" || functions[0].Name() != "local" ||
		functions[1].Pkg().Path() != "strings" || functions[1].Name() != "TrimSpace" {
		t.Fatalf("unexpected exported declarations: %v", functions)
	}
	if err := format.Node(&after, fset, file); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before.Bytes(), after.Bytes()) {
		t.Fatal("export preflight changed the syntax")
	}
}

func TestRejectUnprovenExport(t *testing.T) {
	for _, value := range []string{"func() {}", "fn", "object.method", "generic[int]"} {
		t.Run(value, func(t *testing.T) {
			_, _, err := rewriteSource(t, `package fixture
import "unsafe"
`+exportHelper+`
type receiver struct{}
func (receiver) method() {}
func generic[T any]() {}
var object receiver
var fn = func() {}
var p = __ccgo_fp(`+value+`)
`)
			if err == nil || !strings.Contains(err.Error(), "static function declaration") {
				t.Fatalf("unproven export accepted: %v", err)
			}
		})
	}
}

func TestRejectInferredGenericStaticExportBeforeMutation(t *testing.T) {
	for _, target := range []string{"func(int) int", "namedCallback", "aliasCallback"} {
		t.Run(target, func(t *testing.T) {
			source := `package fixture
import "unsafe"
type namedCallback func(int) int
type aliasCallback = func(int) int
func cFuncPointer[T any](f T) uintptr {
	return *(*uintptr)(unsafe.Pointer(&struct{ f T }{f}))
}
func genericFn[T any](value T) T { return value }
var p = cFuncPointer[` + target + `](genericFn)
`
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "generic-export.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			info := &types.Info{
				Types:     make(map[ast.Expr]types.TypeAndValue),
				Defs:      make(map[*ast.Ident]types.Object),
				Uses:      make(map[*ast.Ident]types.Object),
				Instances: make(map[*ast.Ident]types.Instance),
			}
			pkg, err := (&types.Config{Importer: importer.Default()}).Check("fixture", fset, []*ast.File{file}, info)
			if err != nil {
				t.Fatalf("inferred generic function must be valid Go: %v", err)
			}
			inferred := false
			for id, instance := range info.Instances {
				if id.Name == "genericFn" && instance.TypeArgs.Len() == 1 &&
					types.Identical(instance.TypeArgs.At(0), types.Typ[types.Int]) {
					inferred = true
				}
			}
			if !inferred {
				t.Fatal("fixture did not exercise generic function argument inference")
			}
			var before, after bytes.Buffer
			if err := format.Node(&before, fset, file); err != nil {
				t.Fatal(err)
			}
			if _, err := ExportedFunctions(fset, info, []*ast.File{file}); err == nil || !strings.Contains(err.Error(), "generic") {
				t.Fatalf("inferred generic export passed preflight: %v", err)
			}
			if _, err := Rewrite(fset, pkg, info, []*ast.File{file}); err == nil || !strings.Contains(err.Error(), "generic") {
				t.Fatalf("inferred generic export passed rewrite validation: %v", err)
			}
			if err := format.Node(&after, fset, file); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before.Bytes(), after.Bytes()) {
				t.Fatal("rejected generic export changed the syntax")
			}
		})
	}
}

func TestRejectChangedHelperAndEscapedHelper(t *testing.T) {
	for _, source := range []string{
		`func __ccgo_fp(f interface{}) uintptr { return 1 }; var p = __ccgo_fp(callback)`,
		exportHelper + `var exported = __ccgo_fp; var p = exported(callback)`,
	} {
		_, _, err := rewriteSource(t, `package fixture
import "unsafe"
var _ unsafe.Pointer
func callback() {}
`+source)
		if err == nil {
			t.Fatal("unvalidated helper accepted")
		}
	}
}

func TestRejectExporterTypedIdentityMismatch(t *testing.T) {
	const interfaceBody = `{ type iface [2]uintptr; return (*iface)(unsafe.Pointer(&f))[1] }`
	const genericBody = `{ return *(*uintptr)(unsafe.Pointer(&struct{ f T }{f})) }`
	tests := []struct {
		name   string
		helper string
		call   string
	}{
		{"interface-global-f", `var f any = first; func __ccgo_fp(ignored any) uintptr ` + interfaceBody, `__ccgo_fp(second)`},
		{"interface-function-parameter", `func __ccgo_fp(f func() int) uintptr ` + interfaceBody, `__ccgo_fp(second)`},
		{"interface-nonempty-parameter", `func __ccgo_fp(f interface{ String() string }) uintptr ` + interfaceBody, ""},
		{"interface-variadic", `func __ccgo_fp(f ...any) uintptr ` + interfaceBody, `__ccgo_fp(second)`},
		{"interface-generic", `func __ccgo_fp[T any](f T) uintptr ` + interfaceBody, `__ccgo_fp(second)`},
		{"generic-global-f", `const f = 0; func cFuncPointer[T ~int](ignored T) uintptr ` + genericBody, ""},
		{"generic-nongeneric-helper", `type T = func() int; func cFuncPointer(f T) uintptr ` + genericBody, `cFuncPointer(second)`},
		{"generic-extra-type-parameter", `func cFuncPointer[T, U any](f T) uintptr ` + genericBody, `cFuncPointer[func() int, int](second)`},
		{"generic-variadic", `type T = []any; func cFuncPointer(f ...any) uintptr ` + genericBody, `cFuncPointer(second)`},
		{"generic-interface-instantiation", `func cFuncPointer[T any](f T) uintptr ` + genericBody, `cFuncPointer[any](second)`},
		{"generic-named-interface-instantiation", `type Empty interface{}; func cFuncPointer[T any](f T) uintptr ` + genericBody, `cFuncPointer[Empty](second)`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := `package fixture
import "unsafe"
func first() int { return 1 }
func second() int { return 2 }
` + test.helper
			if test.call != "" {
				source += "\nvar p = " + test.call
			}
			original, err := format.Source([]byte(source))
			if err != nil {
				t.Fatal(err)
			}
			out, stats, err := rewriteSource(t, source)
			if err == nil {
				t.Fatal("exporter with unproven typed identity was accepted")
			}
			if stats != (Stats{}) || !bytes.Equal(bytes.TrimSpace(original), bytes.TrimSpace([]byte(out))) {
				t.Fatal("rejected exporter changed the source or reported adaptations")
			}
		})
	}
}

func TestExporterGlobalShadowOriginalGo(t *testing.T) {
	// The unchanged helper body is not enough: &f refers to the package's
	// interface here. Replacing its call by a box of second changes 1 to 2.
	runRewrittenSource(t, `package main
import "unsafe"
func first() int { return 1 }
func second() int { return 2 }
var f any = first
func __ccgo_fp(ignored any) uintptr {
 type iface [2]uintptr
 return (*iface)(unsafe.Pointer(&f))[1]
}
func main() {
 p := __ccgo_fp(second)
 fn := *(*func() int)(unsafe.Pointer(&struct{ uintptr }{p}))
 if fn() != 1 { panic("global interface must retain first") }
}
`)
}

func TestExporterEmptyInterfaceParameter(t *testing.T) {
	for _, parameter := range []string{"any", "empty", "alias"} {
		t.Run(parameter, func(t *testing.T) {
			_, stats, err := rewriteSource(t, `package fixture
import "unsafe"
type empty interface{}
type alias = any
func __ccgo_fp(f `+parameter+`) uintptr {
 type iface [2]uintptr
 return (*iface)(unsafe.Pointer(&f))[1]
}
func callback() {}
var p = __ccgo_fp(callback)
`)
			if err != nil || stats.Exports != 1 {
				t.Fatalf("valid interface exporter rejected: %+v, %v", stats, err)
			}
		})
	}
}

func TestGenericExporterFunctionInstantiations(t *testing.T) {
	for _, parameter := range []string{"func() int", "namedFunction", "aliasFunction"} {
		t.Run(parameter, func(t *testing.T) {
			_, stats, err := rewriteSource(t, `package fixture
import "unsafe"
type namedFunction func() int
type aliasFunction = func() int
func cFuncPointer[T any](f T) uintptr {
 return *(*uintptr)(unsafe.Pointer(&struct{ f T }{f}))
}
func callback() int { return 42 }
var p = cFuncPointer[`+parameter+`](callback)
`)
			if err != nil || stats.Exports != 1 {
				t.Fatalf("valid function instantiation rejected: %+v, %v", stats, err)
			}
		})
	}
}

func TestUnrelatedUnsafeConversionUnchanged(t *testing.T) {
	out, stats, err := rewriteSource(t, `package fixture
import "unsafe"
func raw(p *uintptr) uintptr { return *(*uintptr)(unsafe.Pointer(p)) }
func ordinary(p unsafe.Pointer) func() { return *(*func())(p) }
`)
	if err != nil || stats != (Stats{}) || !strings.Contains(out, "*(*uintptr)(unsafe.Pointer(p))") ||
		!strings.Contains(out, "*(*func())(p)") {
		t.Fatalf("ordinary unsafe changed: %+v, %v\n%s", stats, err, out)
	}
}

func TestConsumerNilRuntime(t *testing.T) {
	source := `package main
import "unsafe"

// A zero CCGo descriptor must decode to a nil function without dereferencing it.
func decode(p uintptr) func() {
	return *(*func())(unsafe.Pointer(&struct{ uintptr }{p}))
}

func main() {
	if decode(0) != nil {
		panic("zero descriptor is not nil")
	}
}
`
	t.Run("original", func(t *testing.T) { runRewrittenSource(t, source) })
	t.Run("adapted", func(t *testing.T) {
		adapted, stats, err := rewriteSource(t, source)
		if err != nil || stats.Imports != 1 {
			t.Fatalf("nil consumer: %+v, %v", stats, err)
		}
		if !strings.Contains(adapted, "// A zero CCGo descriptor must decode to a nil function without dereferencing it.\nfunc decode") {
			t.Fatalf("consumer comment lost or detached:\n%s", adapted)
		}
		runRewrittenSource(t, adapted)
	})
}

func TestConsumerRuntimePreservesTypedValues(t *testing.T) {
	source := `package main
import "unsafe"
` + exportHelper + `

type callback func(int, ...int) (int, bool)

// sum is shared by ordinary Go and adapted callbacks.
func sum(first int, rest ...int) (int, bool) {
	for _, n := range rest {
		first += n
	}
	return first, true
}

var reads int

func descriptor(p uintptr) uintptr {
	reads++
	return p
}

func decode(p uintptr) callback {
	return *(*callback)(unsafe.Pointer(&struct{ uintptr }{
		// Evaluate the descriptor exactly once.
		descriptor(p),
	}))
}

func main() {
	if decode(0) != nil || reads != 1 {
		panic("nil typed callback or evaluation count")
	}
	p := __ccgo_fp(sum)
	fn := decode(p)
	if n, ok := fn(20, 10, 12); !ok || n != 42 || reads != 2 {
		panic("typed variadic callback")
	}
	if p != __ccgo_fp(sum) {
		panic("static callback identity")
	}
}
`
	t.Run("original", func(t *testing.T) { runRewrittenSource(t, source) })
	t.Run("adapted", func(t *testing.T) {
		adapted, stats, err := rewriteSource(t, source)
		if err != nil || stats.Imports != 1 || stats.Exports != 2 || stats.StaticBoxes != 1 {
			t.Fatalf("typed consumer: %+v, %v", stats, err)
		}
		if !strings.Contains(adapted, "// sum is shared by ordinary Go and adapted callbacks.\nfunc sum") ||
			strings.Count(adapted, "// Evaluate the descriptor exactly once.") != 1 {
			t.Fatalf("comments lost or detached:\n%s", adapted)
		}
		runRewrittenSource(t, adapted)
	})
}

func runRewrittenSource(t *testing.T, source string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "run", path)
	command.WaitDelay = 5 * time.Second
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run function-box consumer: %v\n%s", err, output)
	}
}
