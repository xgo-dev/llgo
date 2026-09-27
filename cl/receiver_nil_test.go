//go:build !llgo

package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/goplus/gogen/packages"
	"github.com/xgo-dev/llvm"
	gossa "golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestPointerMethodReceiverNilChecks(t *testing.T) {
	const source = `package foo
type T struct { x int }
func (*T) M(int) {}
type U struct { pad int; T }
type P struct { *T }
func arg() int { return 1 }
func plain(p *T) { p.M(arg()) }
func expression(p *T) { (*T).M(p, arg()) }
func deref(p *T) { (*p).M(arg()) }
func promoted(p *U) { p.M(arg()) }
func pointer(p *P) { p.M(arg()) }
func deferred(p *T) { defer (*p).M(arg()) }
func started(p *T) { go (*p).M(arg()) }
func bound(p *T) func(int) { return (*p).M }
func shadowed(T *T) { (*T).M(arg()) }
func promotedExpression(p *U) { (*U).M(p, arg()) }
func savedPromotedExpression(p *P) { f := (*P).M; f(p, arg()) }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "receiver.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	ssaPkg, info, err := ssautil.BuildPackage(&types.Config{Importer: packages.NewImporter(fset)}, fset,
		types.NewPackage("foo", "foo"), files, gossa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	checks := CollectReceiverNilChecks(files, nil, info)
	if checks == nil || len(checks.calls) != 7 || len(checks.values) != 8 {
		t.Fatalf("receiver check collection = %#v, want 7 calls and 8 selections", checks)
	}
	if CollectReceiverNilChecks(files, nil) != nil || receiverNeedsAddressCheck(nil) {
		t.Fatal("absent type information must not invent receiver checks")
	}
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	if err := prepareTestSyntax(prog, ssaPkg, files, Options{}); err != nil {
		t.Fatal(err)
	}
	prog.Directives().Freeze()
	pkg, _, err := newPackageEx(prog, nil, nil, nil, ssaPkg, files, nil, false, Options{ReceiverNilChecks: checks})
	if err != nil {
		t.Fatal(err)
	}
	mod := pkg.Module()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"plain", "expression"} {
		ir := mustNamedFunction(t, mod, "foo."+name).String()
		if strings.Contains(ir, "AssertNilDeref") {
			t.Fatalf("legal pointer receiver %s gained a nil check:\n%s", name, ir)
		}
	}
	for _, name := range []string{"deref", "promoted", "deferred", "started", "shadowed"} {
		ir := mustNamedFunction(t, mod, "foo."+name).String()
		argument := strings.Index(ir, "@foo.arg(")
		guard := strings.Index(ir, "AssertNilDeref")
		if argument < 0 || guard < argument {
			t.Fatalf("%s must check its receiver after evaluating arguments:\n%s", name, ir)
		}
	}
	// An embedded-pointer receiver has a real load, unlike address-only
	// promotion. Its guard must dominate that load even when optimization
	// removes the eventual method's unused receiver argument. Once the outer
	// pointer is checked, the derived field address itself needs no second guard.
	pointerIR := mustNamedFunction(t, mod, "foo.pointer").String()
	guard := strings.Index(pointerIR, "AssertNilDeref")
	load := strings.Index(pointerIR, "load ptr")
	if guard < 0 || load < guard {
		t.Fatalf("embedded receiver pointer load precedes its guard:\n%s", pointerIR)
	}
	if strings.Contains(pointerIR, "AssertNilDerefPtr") {
		t.Fatalf("embedded receiver pointer load rechecks its derived field address:\n%s", pointerIR)
	}
	ir := mustNamedFunction(t, mod, "foo.bound").String()
	if !strings.Contains(ir, "AssertNilDeref") {
		t.Fatalf("method-value creation lost its nil check:\n%s", ir)
	}
}

func TestImportedGenericPointerMethodReceiverNilChecks(t *testing.T) {
	fset := token.NewFileSet()
	check := func(path, source string, importer types.Importer) (*types.Package, *types.Info, *ast.File) {
		t.Helper()
		file, err := parser.ParseFile(fset, path+".go", source, 0)
		if err != nil {
			t.Fatal(err)
		}
		info := newLocalityTypeInfo()
		pkg, err := (&types.Config{Importer: importer}).Check(path, fset, []*ast.File{file}, info)
		if err != nil {
			t.Fatal(err)
		}
		return pkg, info, file
	}
	dep, depInfo, depFile := check("example.com/dep", `package dep
type T struct{}
func (*T) M(int) {}
func Call[A any](p *T, arg func() int, _ A) { (*p).M(arg()) }
func Bound[A any](p *T, _ A) func(int) { return (*p).M }
`, nil)
	root, rootInfo, rootFile := check("example.com/root", `package root
import "example.com/dep"
func Invoke(p *dep.T, arg func() int) { dep.Call(p, arg, 1); dep.Bound(p, 1)(arg()) }
`, importerFunc(func(path string) (*types.Package, error) {
		if path == dep.Path() {
			return dep, nil
		}
		return nil, types.Error{Msg: "unexpected import " + path}
	}))
	goProg := gossa.NewProgram(fset, gossa.SanityCheckFunctions|gossa.InstantiateGenerics)
	goProg.CreatePackage(dep, []*ast.File{depFile}, depInfo, true)
	rootSSA := goProg.CreatePackage(root, []*ast.File{rootFile}, rootInfo, true)
	goProg.Build()
	if checks := CollectReceiverNilChecks([]*ast.File{rootFile}, rootInfo); checks != nil {
		t.Fatal("the caller's syntax must not contain the dependency's receiver selections")
	}
	checks := CollectReceiverNilChecks([]*ast.File{rootFile, depFile}, rootInfo, depInfo)
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	// This test owns dependency bodies as well as the root syntax.
	if err := ParsePkgSyntax(prog, fset, dep, []*ast.File{depFile}); err != nil {
		t.Fatal(err)
	}
	prog.PackageDirectives(dep).Bind(depInfo)
	if err := ParsePkgSyntax(prog, fset, root, []*ast.File{rootFile}); err != nil {
		t.Fatal(err)
	}
	prog.PackageDirectives(root).Bind(rootInfo)
	prog.Directives().Freeze()
	pkg, _, err := newPackageEx(prog, nil, nil, nil, rootSSA, []*ast.File{rootFile}, nil, false, Options{ReceiverNilChecks: checks})
	if err != nil {
		t.Fatal(err)
	}
	mod := pkg.Module()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Call", "Bound"} {
		found := false
		for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
			if !strings.HasPrefix(fn.Name(), "example.com/dep."+name+"[") {
				continue
			}
			found = true
			if !strings.Contains(fn.String(), "AssertNilDeref") {
				t.Fatalf("imported generic %s lost its source receiver check:\n%s", name, fn.String())
			}
		}
		if !found {
			t.Fatalf("imported generic %s was not emitted in the caller module:\n%s", name, mod.String())
		}
	}
}

func TestReceiverNilChecksRejectNoPos(t *testing.T) {
	for _, test := range []struct {
		name       string
		invalidate func(*ast.CallExpr, []*ast.SelectorExpr)
		calls      int
		values     int
	}{
		{"valid", func(*ast.CallExpr, []*ast.SelectorExpr) {}, 1, 2},
		{"call", func(call *ast.CallExpr, _ []*ast.SelectorExpr) { call.Lparen = token.NoPos }, 0, 2},
		{"selector", func(_ *ast.CallExpr, selectors []*ast.SelectorExpr) { selectors[0].Sel.NamePos = token.NoPos }, 0, 1},
		{"receiver", func(_ *ast.CallExpr, selectors []*ast.SelectorExpr) {
			selectors[0].X.(*ast.Ident).NamePos = token.NoPos
		}, 0, 1},
		{"bound", func(_ *ast.CallExpr, selectors []*ast.SelectorExpr) { selectors[1].Sel.NamePos = token.NoPos }, 1, 1},
		{"all", func(call *ast.CallExpr, selectors []*ast.SelectorExpr) {
			call.Lparen = token.NoPos
			for _, selector := range selectors {
				selector.Sel.NamePos = token.NoPos
			}
		}, 0, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "receiver.go", `package foo
type T struct{}
func (*T) M() {}
func F(p *T) { p.M(); _ = p.M }
`, 0)
			if err != nil {
				t.Fatal(err)
			}
			info := &types.Info{Selections: make(map[*ast.SelectorExpr]*types.Selection)}
			if _, err := new(types.Config).Check("foo", fset, []*ast.File{file}, info); err != nil {
				t.Fatal(err)
			}
			var call *ast.CallExpr
			var selectors []*ast.SelectorExpr
			ast.Inspect(file, func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.CallExpr:
					call = node
				case *ast.SelectorExpr:
					selectors = append(selectors, node)
				}
				return true
			})
			test.invalidate(call, selectors)
			checks := CollectReceiverNilChecks([]*ast.File{file}, info)
			if test.calls == 0 && test.values == 0 {
				if checks != nil {
					t.Fatalf("positionless selections produced checks: %#v", checks)
				}
				return
			}
			if checks == nil || len(checks.calls) != test.calls || len(checks.values) != test.values {
				t.Fatalf("checks = %#v, want %d calls and %d values", checks, test.calls, test.values)
			}
			for _, table := range []map[token.Pos]receiverNilCheck{checks.calls, checks.values} {
				for pos, check := range table {
					if !pos.IsValid() || !check.pos.IsValid() {
						t.Fatalf("source metadata contains invalid position: key=%v check=%#v", pos, check)
					}
				}
			}
		})
	}
}

func TestReceiverNilDerefChecksWithoutSourceMetadata(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "wrapper.go", `package foo
type T struct{}
func (*T) M() {}
type P struct { *T }
func F(p *P) { (*P).M(p) }
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, _, err := ssautil.BuildPackage(new(types.Config), fset, types.NewPackage("foo", "foo"), []*ast.File{file}, gossa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	protected := 0
	for fn := range ssautil.AllFunctions(pkg.Prog) {
		if !isMethodReceiverWrapper(fn) {
			continue
		}
		_, checks := collectMethodNilDerefChecks(fn, nil)
		protected += len(checks)
	}
	if protected == 0 {
		t.Fatal("promoted pointer wrapper lost its receiver-load checks without source metadata")
	}
}

func TestReceiverNilChecksNonLoadReceivers(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "received.go", `package foo
type T struct{}
func (*T) M() {}
type U struct { T }
func receive(ch <-chan *T) { (<-ch).M() }
func receiveBound(ch <-chan *T) func() { return (<-ch).M }
func local() { var value T; value.M(); _ = value.M }
func promoted(value *U) { value.M() }
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	ssaPkg, info, err := ssautil.BuildPackage(new(types.Config), fset, types.NewPackage("foo", "foo"), files, gossa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	checks := CollectReceiverNilChecks(files, info)
	for _, name := range []string{"receive", "receiveBound", "local"} {
		// A channel receive is also an SSA UnOp, but is not a pointer
		// load. Its result can legally be a nil pointer-method receiver.
		if _, loads := collectMethodNilDerefChecks(ssaPkg.Func(name), checks); len(loads) != 0 {
			t.Fatalf("%s invented receiver pointer loads: %v", name, loads)
		}
	}
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	if err := prepareTestSyntax(prog, ssaPkg, files, Options{}); err != nil {
		t.Fatal(err)
	}
	prog.Directives().Freeze()
	pkg, _, err := newPackageEx(prog, nil, nil, nil, ssaPkg, files, nil, false, Options{ReceiverNilChecks: checks})
	if err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"receive", "receiveBound", "local"} {
		body := mustNamedFunction(t, pkg.Module(), "foo."+name).String()
		if strings.Contains(body, "AssertNilDeref") {
			t.Fatalf("%s must permit a received nil or known non-nil local receiver:\n%s", name, body)
		}
	}
	// An unrelated closure with no source metadata requires neither a
	// receiver check nor a builder. Keep that fast path independent of LLVM.
	new(context).checkBoundMethodReceiver(nil, new(gossa.MakeClosure))
	// Selection.Index exposes a mutable slice. Malformed metadata must be
	// rejected by the existing guard instead of indexing outside the struct.
	for _, selection := range info.Selections {
		indices := selection.Index()
		if len(indices) > 1 {
			saved := indices[0]
			indices[0] = 100
			got := receiverNeedsAddressCheck(selection)
			indices[0] = saved
			if got {
				t.Fatal("invalid embedded-field index produced a receiver check")
			}
			return
		}
	}
	t.Fatal("missing promoted selection for the invalid-index guard")
}
