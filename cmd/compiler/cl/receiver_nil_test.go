package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	"github.com/xgo-dev/llgo/cmd/compiler/ssaopt"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssautil"
	"github.com/xgo-dev/llgo/types"
	"github.com/xgo-dev/llvm"
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
	ssaPkg, _, err := ssautil.BuildPackage(&types.Config{Importer: packages.NewImporter(fset)}, fset,
		types.NewPackage("foo", "foo"), files, ssa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	pkg, _, err := newPackageEx(prog, nil, nil, nil, ssaPkg, files, nil, false, Options{})
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
	goProg := ssa.NewProgram(fset, ssa.SanityCheckFunctions|ssa.InstantiateGenerics)
	goProg.CreatePackage(dep, []*ast.File{depFile}, depInfo, true)
	rootSSA := goProg.CreatePackage(root, []*ast.File{rootFile}, rootInfo, true)
	goProg.Build()
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	pkg, _, err := newPackageEx(prog, nil, nil, nil, rootSSA, []*ast.File{rootFile}, nil, false, Options{})
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
	pkg, _, err := ssautil.BuildPackage(new(types.Config), fset, types.NewPackage("foo", "foo"), []*ast.File{file}, ssa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	protected := 0
	for fn := range ssautil.AllFunctions(pkg.Prog) {
		if !isMethodReceiverWrapper(fn) {
			continue
		}
		plan, err := ssaopt.PlanFunction(fn, ssaopt.Config{Verify: true})
		if err != nil {
			t.Fatalf("PlanFunction(%s): %v", fn, err)
		}
		protected += len(plan.RecvNilDerefChecks)
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
	ssaPkg, _, err := ssautil.BuildPackage(new(types.Config), fset, types.NewPackage("foo", "foo"), files, ssa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"receive", "receiveBound", "local"} {
		// A channel receive is also an SSA UnOp, but is not a pointer
		// load. Its result can legally be a nil pointer-method receiver.
		plan, err := ssaopt.PlanFunction(ssaPkg.Func(name), ssaopt.Config{Verify: true})
		if err != nil {
			t.Fatalf("PlanFunction(%s): %v", name, err)
		}
		if len(plan.RecvNilDerefChecks) != 0 {
			t.Fatalf("%s invented receiver pointer loads: %v", name, plan.RecvNilDerefChecks)
		}
	}
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	pkg, _, err := newPackageEx(prog, nil, nil, nil, ssaPkg, files, nil, false, Options{})
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
	// An unrelated closure with no optPlan requires neither a receiver check nor a builder.
	new(context).checkBoundMethodReceiver(nil, new(ssa.MakeClosure))
}
