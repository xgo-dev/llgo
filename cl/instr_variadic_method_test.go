package cl

import (
	"errors"
	"go/types"
	"os"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/cabi"
	"github.com/xgo-dev/llgo/internal/optlevel"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
)

func TestCVariadicWrapperCycles(t *testing.T) {
	const source = `package foo
type T int32
func (t *T) A(__llgo_va_list ...any) int32 {
	t.C(__llgo_va_list...)
	return t.B(__llgo_va_list...)
}
func (t *T) B(__llgo_va_list ...any) int32 { return t.A(__llgo_va_list...) }
func (t *T) C(__llgo_va_list ...any) int32 { return 0 }
`
	ssaPkg, _, _ := buildGoSSAPkg(t, source)
	methods := types.NewMethodSet(types.NewPointer(ssaPkg.Pkg.Scope().Lookup("T").Type()))
	wrappers := make(map[string]*ssa.Function)
	for _, name := range []string{"A", "B", "C"} {
		fn := ssaPkg.Prog.MethodValue(methods.Lookup(ssaPkg.Pkg, name))
		fn.Synthetic = "wrapper for " + name
		wrappers[name] = fn
	}
	prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	p := &context{prog: prog}
	checked := make(map[*ssa.Function]error)
	if err := p.cVariadicWrapperError(wrappers["A"], checked); err != nil {
		t.Fatalf("supported wrapper cycle did not terminate: %v", err)
	}
	for name, fn := range wrappers {
		if err, ok := checked[fn]; !ok || err != nil {
			t.Fatalf("supported wrapper %s was not memoized: %v", name, err)
		}
	}
	// An exit from a cycle can still be unsupported. Do not memoize a
	// back-edge as a successful complete traversal and hide that exit.
	want := errors.New("unsupported forwarding at cycle exit")
	p.variadicErrors = map[*ssa.Function]error{wrappers["C"]: want}
	checked = make(map[*ssa.Function]error)
	for _, name := range []string{"A", "B", "A"} {
		if err := p.cVariadicWrapperError(wrappers[name], checked); !errors.Is(err, want) {
			t.Fatalf("wrapper %s lost the cycle's unsupported exit: %v", name, err)
		}
	}
}

func TestCVariadicMethodWrapperForwarding(t *testing.T) {
	source, err := os.ReadFile("_testrt/vamethod/in.go")
	if err != nil {
		t.Fatal(err)
	}
	ssaPkg, _, files := buildGoSSAPkg(t, string(source))
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "arm"},
		{GOOS: "linux", GOARCH: "riscv64"},
		{GOOS: "darwin", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "amd64", LLVMTarget: "x86_64-pc-windows-msvc"},
		{GOOS: "windows", GOARCH: "amd64", LLVMTarget: "x86_64-w64-windows-gnu"},
		{GOOS: "windows", GOARCH: "arm64", LLVMTarget: "aarch64-pc-windows-msvc"},
		{GOOS: "windows", GOARCH: "arm64", LLVMTarget: "aarch64-w64-windows-gnu"},
		{GOOS: "windows", GOARCH: "386", LLVMTarget: "i686-pc-windows-msvc"},
		{GOOS: "windows", GOARCH: "386", LLVMTarget: "i686-w64-windows-gnu"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH+"/"+target.LLVMTarget, func(t *testing.T) {
			target.OptLevel = optlevel.O0
			prog := newLLSSAProgForTarget(t, &target)
			defer prog.Dispose()
			pkg, err := NewPackage(prog, ssaPkg, files)
			if err != nil {
				t.Fatal(err)
			}
			mod := pkg.Module()
			cabi.NewTransformer(prog, target.LLVMTarget, "", false).TransformModule(pkg.Path(), mod)
			fn := mustNamedFunction(t, mod, "main.(*CFmt).Printf")
			if text := fn.String(); !strings.Contains(text, "musttail call i32 (ptr, ...) @printf(ptr ") ||
				!strings.Contains(text, ", ...)") {
				t.Fatalf("wrapper does not forward its incoming ellipsis:\n%s", text)
			}
			if fn.GetStringAttributeAtIndex(-1, "thunk").IsNil() {
				t.Fatal("variadic forwarding requires the LLVM thunk attribute")
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			object, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.ObjectFile)
			if err != nil {
				t.Fatal(err)
			}
			object.Dispose()
		})
	}
}

func TestCVariadicMethodExternalDeclaration(t *testing.T) {
	// An external declaration has no Go body to prove that it cannot recover.
	// The forwarding thunk must still tail-call it without a post-call alias
	// cleanup, and preserve its extra fixed arguments and void result.
	const source = `package foo
import _ "unsafe"
type T int8
//go:linkname (*T).Check C.check
func (*T) Check(tag int32, number float64, data *int8, __llgo_va_list ...any)
type Wrapper struct { *T }
func use(w *Wrapper) { w.Check(7, 1.5, new(int8), int32(42)) }
`
	ssaPkg, _, files := buildGoSSAPkg(t, source)
	prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64", OptLevel: optlevel.O0})
	defer prog.Dispose()
	pkg, err := NewPackage(prog, ssaPkg, files)
	if err != nil {
		t.Fatal(err)
	}
	fn := mustNamedFunction(t, pkg.Module(), "foo.(*Wrapper).Check")
	if text := fn.String(); !strings.Contains(text, "musttail call void (ptr, i32, double, ptr, ...) @check(") ||
		strings.Contains(text, "RecoverFrameAlias") {
		t.Fatalf("external variadic declaration is not tail-forwarded:\n%s", text)
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestCVariadicValueReceiverWrapperIR(t *testing.T) {
	const source = `package foo
type T int32
//llgo:link T.Check C.check
func (T) Check(__llgo_va_list ...any) int32 { return 0 }
type Wrapper struct { T }
func use(w *Wrapper) int32 { return w.Check(int32(42)) }
`
	ssaPkg, _, files := buildGoSSAPkg(t, source)
	prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	pkg, err := NewPackage(prog, ssaPkg, files)
	if err != nil {
		t.Fatal(err)
	}
	// Loading a scalar receiver changes the fixed ABI from ptr to i32. Such
	// wrappers cannot use LLVM's variadic musttail forwarding mechanism.
	// This direct call supplies explicit arguments. Its unused synthetic
	// wrapper must remain valid IR; interface uses are rejected separately.
	fn := mustNamedFunction(t, pkg.Module(), "foo.(*Wrapper).Check")
	if text := fn.String(); !strings.Contains(text, "call i32 (i32, ...) @check(") || strings.Contains(text, "musttail") {
		t.Fatalf("excluded receiver ABI unexpectedly changed lowering:\n%s", text)
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestCVariadicIncompatibleSignatureIR(t *testing.T) {
	// Ordinary methods have no ellipsis to forward. Aggregate signatures need
	// instructions or ABI rewrites that cannot retain a musttail call followed
	// immediately by its return. Direct calls have explicit arguments and
	// remain supported; their unused wrappers need valid IR without musttail.
	for _, tc := range []struct {
		name, declaration, use string
	}{
		{"non_variadic",
			"func (*T) Check(value int32) int32 { return 0 }",
			"func use(w *Wrapper) int32 { return w.Check(42) }"},
		{"multiple_results",
			"func (*T) Check(__llgo_va_list ...any) (int32, int32) { return 0, 0 }",
			"func use(w *Wrapper) (int32, int32) { return w.Check(int32(42)) }"},
		{"aggregate_result",
			"func (*T) Check(__llgo_va_list ...any) R { return R{} }",
			"func use(w *Wrapper) R { return w.Check(int32(42)) }"},
		{"aggregate_parameter",
			"func (*T) Check(tag R, __llgo_va_list ...any) int32 { return 0 }",
			"func use(w *Wrapper) int32 { return w.Check(R{}, int32(42)) }"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := "package foo\ntype T int8\ntype R struct { a, b int64 }\n" +
				"//llgo:link (*T).Check C.check\n" + tc.declaration +
				"\ntype Wrapper struct { *T }\n" + tc.use
			ssaPkg, _, files := buildGoSSAPkg(t, source)
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
			defer prog.Dispose()
			pkg, err := NewPackage(prog, ssaPkg, files)
			if err != nil {
				t.Fatal(err)
			}
			mod := pkg.Module()
			fn := mustNamedFunction(t, mod, "foo.(*Wrapper).Check")
			if text := fn.String(); !strings.Contains(text, "@check(") || strings.Contains(text, "musttail") {
				t.Fatalf("excluded wrapper unexpectedly changed lowering:\n%s", text)
			}
			if !fn.GetStringAttributeAtIndex(-1, "thunk").IsNil() {
				t.Fatal("incompatible wrapper must not be marked as a forwarding thunk")
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			cabi.NewTransformer(prog, "", "", false).TransformModule(pkg.Path(), mod)
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCVariadicUnrelatedInterfaces(t *testing.T) {
	for _, use := range []struct{ name, source string }{
		{"value_any", "func use(v T) any { return v }"},
		{"pointer_any", "func use(v *T) any { return v }"},
		{"value_stringer", "func use(v T) Stringer { return v }"},
		{"pointer_stringer", "func use(v *T) Stringer { return v }"},
		{"promoted_any", "func use(v Wrapper) any { return v }"},
		{"promoted_stringer", "func use(v *Wrapper) Stringer { return v }"},
		{"embedded_interface", "func use(v Wrapper) Embedded { return v }"},
		{"interface_alias", "func use(v T) Alias { return v }"},
		{"global", "var value Stringer = T(7)"},
		{"generic", "func expose[V any](v V) any { return v }; func use(v T) any { return expose(v) }"},
	} {
		for _, target := range []llssa.Target{
			{GOOS: "linux", GOARCH: "amd64"},
			{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"},
		} {
			t.Run(use.name+"/"+target.GOOS, func(t *testing.T) {
				const declarations = `package foo
type T int32
//llgo:link T.Check C.check
func (T) Check(__llgo_va_list ...any) int32 { return 0 }
func (T) String() string { return "ok" }
type Wrapper struct { T }
type Stringer interface { String() string }
type Embedded interface { Stringer }
type Alias = Stringer
`
				ssaPkg, _, files := buildGoSSAPkg(t, declarations+use.source)
				prog := newLLSSAProgForTarget(t, &target)
				defer prog.Dispose()
				pkg, err := NewPackage(prog, ssaPkg, files)
				if err != nil {
					t.Fatalf("unrelated interface exposed the C variadic method: %v", err)
				}
				if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestCVariadicUnsupportedWrapperUses(t *testing.T) {
	for _, tc := range []struct {
		name, declaration, use, reason string
	}{
		{"value_receiver", "func (T) Check(__llgo_va_list ...any) int32 { return 0 }",
			"func use(w *Wrapper) Checker { return w }", "pointer receivers"},
		{"boxed_value_receiver", "func (T) Check(__llgo_va_list ...any) int32 { return 0 }",
			"func use(w Wrapper) Checker { return w }", "pointer receivers"},
		{"nested_value_receiver", "func (T) Check(__llgo_va_list ...any) int32 { return 0 }",
			"type Nested struct { *Wrapper }; func use(w *Wrapper) Checker { return &Nested{w} }", "pointer receivers"},
		{"generic_interface_conversion", "func (T) Check(__llgo_va_list ...any) int32 { return 0 }",
			"func expose[V any](w *Wrapper, v V) Checker { return w }; func use(w *Wrapper) Checker { return expose(w, 0) }", "pointer receivers"},
		{"global_interface_conversion", "func (T) Check(__llgo_va_list ...any) int32 { return 0 }",
			"var value Checker = &Wrapper{}", "pointer receivers"},
		{"aggregate_parameter", "func (*T) Check(tag R, __llgo_va_list ...any) int32 { return 0 }",
			"func use(w *Wrapper) Checker { return w }", "fixed parameter"},
		{"aggregate_result", "func (*T) Check(__llgo_va_list ...any) R { return R{} }",
			"func use(w *Wrapper) Checker { return w }", "result"},
		{"multiple_results", "func (*T) Check(__llgo_va_list ...any) (int32, int32) { return 0, 0 }",
			"func use(w *Wrapper) Checker { return w }", "multiple results"},
		{"bound_method", "func (*T) Check(__llgo_va_list ...any) int32 { return 0 }",
			"func use(w *Wrapper) { fn := w.Check; fn(int32(42)) }", "pointer receivers"},
		{"method_expression", "func (T) Check(__llgo_va_list ...any) int32 { return 0 }",
			"func use() { fn := Wrapper.Check; fn(Wrapper{}, int32(42)) }", "pointer receivers"},
		{"bound_interface_method", "func (*T) Check(__llgo_va_list ...any) int32 { return 0 }",
			"func use(w Checker) { fn := w.Check; fn(int32(42)) }", "pointer receivers"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			method := strings.TrimPrefix(tc.declaration, "func ")
			method = method[strings.Index(method, ") ")+2:]
			method = method[:strings.Index(method, " {")]
			source := "package foo\ntype T int32\ntype R struct { a, b int64 }\n" +
				"//llgo:link T.Check C.check\n" + tc.declaration +
				"\ntype Wrapper struct { T }\ntype Checker interface { " + method + " }\n" + tc.use
			if strings.Contains(tc.declaration, "(*T)") {
				source = strings.Replace(source, "//llgo:link T.Check", "//llgo:link (*T).Check", 1)
			}
			ssaPkg, _, files := buildGoSSAPkg(t, source)
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
			defer prog.Dispose()
			_, err := NewPackage(prog, ssaPkg, files)
			if err == nil {
				t.Fatal("unsupported incoming ellipsis was silently accepted")
			}
			for _, want := range []string{"cannot forward C variadic arguments", "Check", "linux/amd64", tc.reason} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("diagnostic = %v, want %q", err, want)
				}
			}
		})
	}
}

func TestCVariadicWrapperTargetDiagnostics(t *testing.T) {
	const source = `package foo
type T int32
//llgo:link (*T).Check C.check
func (*T) Check(__llgo_va_list ...any) int32 { return 0 }
type Wrapper struct { *T }
type Checker interface { Check(__llgo_va_list ...any) int32 }
func use(w *Wrapper) Checker { return w }
`
	for _, target := range []llssa.Target{
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"},
		{GOOS: "js", GOARCH: "wasm", WasmProfile: "j64"},
		{GOOS: "linux", GOARCH: "amd64", Target: "custom_native"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH+"/"+target.Target, func(t *testing.T) {
			ssaPkg, _, files := buildGoSSAPkg(t, source)
			prog := newLLSSAProgForTarget(t, &target)
			defer prog.Dispose()
			_, err := NewPackage(prog, ssaPkg, files)
			if err == nil {
				t.Fatal("unqualified target silently accepted incoming C ellipsis")
			}
			for _, want := range []string{"cannot forward C variadic arguments", "Check", target.GOOS + "/" + target.GOARCH} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("diagnostic = %v, want %q", err, want)
				}
			}
			if target.Target != "" && !strings.Contains(err.Error(), target.Target) {
				t.Fatalf("diagnostic omitted explicit target profile: %v", err)
			}
		})
	}
}

func TestCVariadicDirectCallsOnUnqualifiedTargets(t *testing.T) {
	const source = `package foo
type T int32
//llgo:link T.Check C.check
func (T) Check(__llgo_va_list ...any) int32 { return 0 }
type Wrapper struct { T }
func use(w *Wrapper) int32 { return w.Check(int32(100), int32(200)) }
`
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"},
		{GOOS: "linux", GOARCH: "amd64", Target: "custom_native"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH+"/"+target.Target, func(t *testing.T) {
			ssaPkg, _, files := buildGoSSAPkg(t, source)
			prog := newLLSSAProgForTarget(t, &target)
			defer prog.Dispose()
			pkg, err := NewPackage(prog, ssaPkg, files)
			if err != nil {
				t.Fatalf("explicit C arguments do not require variadic forwarding: %v", err)
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
