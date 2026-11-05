package cl

import (
	"fmt"
	"go/types"
	"strings"
	"sync"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	gossa "golang.org/x/tools/go/ssa"
)

func TestCrossPackageInlineBodies(t *testing.T) {
	dep, root := buildCallerFrameSSAProgram(t, "example.com/dep", `package dep
func add(x, y uint64) uint64 { return x + y }
func Sum(x, y uint64) uint64 { return add(x, y) + 2 }
func Split(x uint64) (uint64, bool) { return x + 1, x > 3 }
func Branch(x int) int { if x < 0 { return -x }; return x + 5 }
//go:noinline
func Kept(x uint64) uint64 { return x + 1 }
func Recursive(x uint64) uint64 { if x == 0 { return 0 }; return Recursive(x-1) }
`, "example.com/root", `package root
import "example.com/dep"
func Sum(x, y uint64) uint64 { return dep.Sum(x, y) }
func Split(x uint64) (uint64, bool) { return dep.Split(x) }
func Branch(x int) int { return dep.Branch(x) }
func Kept(x uint64) uint64 { return dep.Kept(x) }
func Recursive(x uint64) uint64 { return dep.Recursive(x) }
var Value = dep.Sum
`)
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "wasip1", GOARCH: "wasm"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			prog := newLLSSAProgForTarget(t, &target)
			defer prog.Dispose()
			tracking := NewCallerTracking()
			tracking.Precompute([]*gossa.Package{dep, root})
			owner, _, err := NewPackageExWithEmbed(prog, tracking, nil, nil, dep, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer owner.Module().Dispose()
			pkg, _, err := NewPackageExWithEmbed(prog, tracking, nil, nil, root, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			mod := pkg.Module()
			defer mod.Dispose()
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"Sum", "Split", "Branch", "add"} {
				fn := mustNamedFunction(t, mod, "example.com/dep."+name)
				if fn.IsDeclaration() || fn.Linkage() != llvm.AvailableExternallyLinkage {
					t.Fatalf("missing optimization-only definition: %s", fn.String())
				}
			}
			for _, name := range []string{"Kept", "Recursive"} {
				if !mustNamedFunction(t, mod, "example.com/dep."+name).IsDeclaration() {
					t.Fatalf("ineligible body %s imported", name)
				}
			}
			if fn := mustNamedFunction(t, owner.Module(), "example.com/dep.Sum"); fn.IsDeclaration() || fn.Linkage() != llvm.ExternalLinkage {
				t.Fatal("import changed defining package's real entry")
			}
			if !strings.Contains(mod.NamedGlobal("example.com/root.Value").String(), "example.com/dep.Sum") {
				t.Fatal("function value lost canonical entry")
			}
			mod.SetDataLayout(prog.DataLayout())
			mod.SetTarget(target.Spec().Triple)
			opts := llvm.NewPassBuilderOptions()
			defer opts.Dispose()
			opts.SetVerifyEach(true)
			if err := mod.RunPasses("default<O2>", prog.TargetMachine(), opts); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"Sum", "Split", "Branch"} {
				if ir := mustNamedFunction(t, mod, "example.com/root."+name).String(); strings.Contains(ir, "call ") {
					t.Fatalf("cross-package call survived optimization:\n%s", ir)
				}
			}
			for _, name := range []string{"Kept", "Recursive"} {
				if ir := mustNamedFunction(t, mod, "example.com/root."+name).String(); !strings.Contains(ir, "example.com/dep."+name) {
					t.Fatalf("ineligible call disappeared:\n%s", ir)
				}
			}
			buf, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
			if err != nil {
				t.Fatal(err)
			}
			buf.Dispose()
		})
	}
}

func TestCrossPackageInlineCostPolicy(t *testing.T) {
	large := "func Large(x int) int {\n"
	for i := 1; i <= inlineBodyBudget; i++ {
		large += fmt.Sprintf("x = x*3 + %d\n", i)
	}
	large += "return x }\n"
	pkg, _, _ := buildGoSSAPkg(t, `package policy
func Small(x int) int { return x + 1 }
func Twice(x int) int { return Small(x) + Small(x+1) }
//go:noinline
func Kept(x int) int { return x + 1 }
func Recursive(x int) int { if x == 0 { return 0 }; return Recursive(x-1) }
func A(x int) int { return B(x) }
func B(x int) int { return A(x) }
func Loop(x int) int { for x > 0 { x-- }; return x }
func Defers(x int) int { defer Small(x); return x }
func Recover() any { return recover() }
func Closure(x int) func() int { return func() int { return x } }
`+large)
	facts := newInlineFacts()
	for _, name := range []string{"Kept", "Recursive", "A", "B", "Loop", "Defers", "Recover", "Closure", "Large"} {
		if cost := facts.cost(pkg.Func(name)); cost <= inlineBodyBudget {
			t.Errorf("%s accepted with cost %d", name, cost)
		}
	}
	if cost := facts.cost(pkg.Func("Small")); cost > inlineBodyBudget {
		t.Fatal("small function rejected")
	}
	if cost := facts.cost(pkg.Func("Twice")); cost > inlineBodyBudget {
		t.Fatal("small repeated callee rejected")
	}
	count := len(facts.costs)
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, name := range []string{"Small", "Kept", "Large"} {
				facts.cost(pkg.Func(name))
			}
		}()
	}
	wg.Wait()
	if len(facts.costs) != count {
		t.Fatal("cached queries added analyses")
	}
}

func TestCrossPackageInlinePointerAndBuiltin(t *testing.T) {
	dep, root := buildCallerFrameSSAProgram(t, "example.com/dep", `package dep
type S struct { X int }
func Address(p *S) *int {return &p.X}
func Size(s string) int {return len(s)}
func Apply(fn func(int)int,x int) int {return fn(x)}
`, "example.com/root", `package root
import "example.com/dep"
func Address(p *dep.S)*int{return dep.Address(p)}
func Size(s string)int{return dep.Size(s)}
`)
	if newInlineFacts().cost(dep.Func("Apply")) > inlineBodyBudget {
		t.Fatal("single indirect call exceeds body budget")
	}
	prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	pkg, err := NewPackage(prog, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer pkg.Module().Dispose()
	address := mustNamedFunction(t, pkg.Module(), "example.com/dep.Address")
	if address.IsDeclaration() || !strings.Contains(address.String(), "AssertNilDeref") {
		t.Fatalf("import lost explicit field-address nil check:\n%s", address.String())
	}
	if mustNamedFunction(t, pkg.Module(), "example.com/dep.Size").IsDeclaration() {
		t.Fatal("builtin wrapper not imported")
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestCrossPackageInlinePatchGuards(t *testing.T) {
	for _, noinline := range []bool{false, true} {
		t.Run(fmt.Sprint(noinline), func(t *testing.T) {
			pragma := ""
			if noinline {
				pragma = "//go:noinline\n"
			}
			dep, root := buildCallerFrameSSAProgram(t, "example.com/dep", "package dep\n"+pragma+"func F(x int)int{return x+1}\n",
				"example.com/root", "package root\nimport \"example.com/dep\"\nfunc Use(x int)int{return dep.F(x)}\n")
			params := "x,y int"
			if noinline {
				params = "x int"
			}
			alt, _, _ := buildGoSSAPkg(t, "package dep\nfunc F("+params+")int{return x+2}\n")
			alt.Pkg = dep.Pkg
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
			defer prog.Dispose()
			pkg := prog.NewPackage("root", "example.com/root")
			defer pkg.Module().Dispose()
			ctx := &context{prog: prog, pkg: pkg, goPkg: root, goTyps: root.Pkg, goProg: root.Prog,
				patches: Patches{dep.Pkg.Path(): {Alt: alt, Types: dep.Pkg}}, inlineFacts: newInlineFacts(),
				loaded: map[*types.Package]*pkgInfo{dep.Pkg: {kind: PkgNormal}}}
			ctx.importInlineBody(dep.Func("F"))
			ctx.importInlineBody(dep.Func("F"))
			if len(ctx.inits) != 0 {
				t.Fatal("incompatible or noinline source patch imported")
			}
		})
	}
}

func TestCrossPackageInlineBoundsDeepCycleAnalysis(t *testing.T) {
	src := "package cycle\n"
	for i := 0; i < 40; i++ {
		src += fmt.Sprintf("func F%d(x int)int{return F%d(x)}\n", i, (i+1)%40)
	}
	pkg, _, _ := buildGoSSAPkg(t, src)
	facts := newInlineFacts()
	if facts.cost(pkg.Func("F0")) <= inlineBodyBudget || facts.cost(pkg.Func("F39")) <= inlineBodyBudget {
		t.Fatal("deep cycle accepted")
	}
}

func TestCrossPackageInlinePreservesExistingDefinition(t *testing.T) {
	dep, root := buildCallerFrameSSAProgram(t, "example.com/dep", "package dep\nfunc F(x int)int{return x+1}\n",
		"example.com/root", "package root\nimport \"example.com/dep\"\nfunc Use(x int)int{return dep.F(x)}\n")
	for _, late := range []bool{false, true} {
		t.Run(fmt.Sprint(late), func(t *testing.T) {
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
			defer prog.Dispose()
			pkg := prog.NewPackage("root", "example.com/root")
			defer pkg.Module().Dispose()
			ctx := &context{prog: prog, pkg: pkg, goPkg: root, goTyps: root.Pkg, goProg: root.Prog,
				inlineFacts: newInlineFacts(), loaded: map[*types.Package]*pkgInfo{dep.Pkg: {kind: PkgNormal}}}
			if late {
				ctx.importInlineBody(dep.Func("F"))
			}
			entry := pkg.NewFunc("example.com/dep.F", dep.Func("F").Signature, llssa.GoDecl)
			b := entry.MakeBody(1)
			b.Return(entry.Param(0))
			b.EndBuild()
			b.Dispose()
			before := pkg.Module().NamedFunction(entry.Name()).String()
			if !late {
				ctx.importInlineBody(dep.Func("F"))
			}
			for _, init := range ctx.inits {
				init()
			}
			if after := pkg.Module().NamedFunction(entry.Name()).String(); after != before {
				t.Fatalf("existing definition changed:\n%s", after)
			}
		})
	}
}

func TestCrossPackageInlineDisabledAndDepth(t *testing.T) {
	src := "package dep\nfunc F6(x int) int {return x+1}\n"
	for i := 5; i >= 0; i-- {
		src += fmt.Sprintf("func F%d(x int) int {return F%d(x)}\n", i, i+1)
	}
	dep, root := buildCallerFrameSSAProgram(t, "example.com/dep", src,
		"example.com/root", "package root\nimport \"example.com/dep\"\nfunc Use(x int) int {return dep.F0(x)}\n")
	tracking := NewCallerTracking()
	tracking.Precompute([]*gossa.Package{dep, root})
	// Prewarm an accepted result; disabling inlining must still prevent import.
	tracking.inlines.cost(dep.Func("F0"))
	for _, disabled := range []bool{false, true} {
		t.Run(fmt.Sprint(disabled), func(t *testing.T) {
			old := disableInline
			disableInline = disabled
			defer func() { disableInline = old }()
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
			defer prog.Dispose()
			pkg, _, err := NewPackageExWithEmbed(prog, tracking, nil, nil, root, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			mod := pkg.Module()
			defer mod.Dispose()
			first := mustNamedFunction(t, mod, "example.com/dep.F0")
			if first.IsDeclaration() != disabled {
				t.Fatal("incorrect import with disabled inlining")
			}
			if !disabled {
				last := mustNamedFunction(t, mod, fmt.Sprintf("example.com/dep.F%d", inlineImportDepth))
				if !last.IsDeclaration() {
					t.Fatal("import depth budget exceeded")
				}
			}
		})
	}
}

func TestCrossPackageInlineModuleBudget(t *testing.T) {
	depSrc, rootSrc := "package dep\n", "package root\nimport \"example.com/dep\"\nfunc Use(x int) int {return "
	const count = 80
	for i := 0; i < count; i++ {
		depSrc += fmt.Sprintf("func F%d(x int) int {", i)
		for n := 1; n <= 10; n++ {
			depSrc += fmt.Sprintf("x=x*3+%d;", i+n)
		}
		depSrc += "return x}\n"
		if i != 0 {
			rootSrc += "+"
		}
		rootSrc += fmt.Sprintf("dep.F%d(x)", i)
	}
	rootSrc += "}\n"
	dep, root := buildCallerFrameSSAProgram(t, "example.com/dep", depSrc, "example.com/root", rootSrc)
	prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	pkg, err := NewPackage(prog, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer pkg.Module().Dispose()
	facts := newInlineFacts()
	imported, cost := 0, 0
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("F%d", i)
		if !mustNamedFunction(t, pkg.Module(), "example.com/dep."+name).IsDeclaration() {
			imported++
			cost += facts.cost(dep.Func(name))
		}
	}
	if imported == 0 || imported == count || cost > inlineModuleBudget {
		t.Fatalf("imported %d bodies at cost %d", imported, cost)
	}
}
