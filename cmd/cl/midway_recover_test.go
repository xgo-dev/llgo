package cl

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestMidwayRecoverDispatcherRequiresEveryVariant(t *testing.T) {
	for _, widths := range [][]int{{0, 128}, {0, 128, 256}, {0, 256}, {128, 256}, {256}} {
		for _, missing := range append([]int{-1}, widths...) {
			t.Run(fmt.Sprintf("widths=%v/missing=%d", widths, missing), func(t *testing.T) {
				var declarations, calls strings.Builder
				for _, width := range widths {
					fmt.Fprintf(&declarations, "func variant%d() {}\n", width)
					if width != missing {
						// Duplicate calls must not stand in for a missing variant.
						fmt.Fprintf(&calls, "variant%d(); variant%d();", width, width)
					}
				}
				fs := token.NewFileSet()
				file, err := parser.ParseFile(fs, "dispatch.go", "package fixture\n"+declarations.String()+"func entry(){"+calls.String()+"}", 0)
				if err != nil {
					t.Fatal(err)
				}
				ast.Inspect(file, func(n ast.Node) bool {
					if id, ok := n.(*ast.Ident); ok && strings.HasPrefix(id.Name, "variant") {
						id.Name = "entry@simd" + strings.TrimPrefix(id.Name, "variant")
					}
					return true
				})
				pkg, _, err := ssautil.BuildPackage(&types.Config{}, fs, types.NewPackage("fixture", "fixture"), []*ast.File{file}, ssa.InstantiateGenerics)
				if err != nil {
					t.Fatal(err)
				}
				want := missing == -1 && pkg.Func("entry@simd0") != nil && pkg.Func("entry@simd128") != nil
				if got := isMidwayRecoverDispatcher(pkg.Func("entry")); got != want {
					t.Fatalf("dispatcher = %v, want %v", got, want)
				}
			})
		}
	}
}

func TestMidwayRecoverDispatcherIdentity(t *testing.T) {
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, "dispatch.go", `package fixture
 func software[T any]() {recover()}
 func hardware[T any]() {recover()}
 func wider[T any]() {recover()}
 func entry[T any](emulated bool) {if emulated {software[T]();return};hardware[T]()}
 func wideEntry[T any](width int) {if width==0 {wideSoftware[T]();return};if width==256 {wider[T]();return};wideHardware[T]()}
 func wideSoftware[T any]() {recover()}
 func wideHardware[T any]() {recover()}
 func trapSoftware() {}
 func trapHardware() {}
 func trap() {panic("unspecialized")}
 func partialSoftware() {}
 func partialHardware() {}
 func partial(fn func()) {fn();partialSoftware()}
 func partialWideSoftware() {recover()}
 func partialWideHardware() {recover()}
 func partialWideWider() {recover()}
 func partialWide(emulated bool) {if emulated {partialWideSoftware();return};partialWideHardware()}
 func ordinary() {entry[int](false)}
 func use() {entry[int](true);wideEntry[int](256);ordinary();trap();partial(nil);partialWide(false)}
 `, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Mirror the compiler-only names which cannot be written in Go source.
	names := map[string]string{
		"software": "entry@simd0", "hardware": "entry@simd128",
		"wideSoftware": "wideEntry@simd0", "wideHardware": "wideEntry@simd128", "wider": "wideEntry@simd256",
		"trapSoftware": "trap@simd0", "trapHardware": "trap@simd128",
		"partialSoftware": "partial@simd0", "partialHardware": "partial@simd128",
		"partialWideSoftware": "partialWide@simd0", "partialWideHardware": "partialWide@simd128", "partialWideWider": "partialWide@simd256",
	}
	ast.Inspect(file, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			if name := names[id.Name]; name != "" {
				id.Name = name
			}
		}
		return true
	})
	pkg, _, err := ssautil.BuildPackage(&types.Config{}, fs, types.NewPackage("fixture", "fixture"), []*ast.File{file}, ssa.InstantiateGenerics)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for fn := range ssautil.AllFunctions(pkg.Prog) {
		if strings.HasPrefix(fn.Name(), "entry[") || strings.HasPrefix(fn.Name(), "wideEntry[") {
			count++
			if !isRecoverTransparentWrapper(fn) || !newRecoverFacts().needsRecoverScope(fn) {
				t.Fatalf("generic dispatcher lost recover identity: %s", fn)
			}
		}
	}
	if count != 2 {
		t.Fatalf("found %d instantiated dispatchers, want 2", count)
	}
	for _, name := range []string{"ordinary", "trap", "partial", "partialWide"} {
		if isRecoverTransparentWrapper(pkg.Func(name)) {
			t.Fatalf("%s must keep its own recovery frame", name)
		}
	}
	if isRecoverTransparentWrapper(nil) || isRecoverTransparentWrapper(&ssa.Function{}) {
		t.Fatal("invalid function classified as dispatcher")
	}
}
