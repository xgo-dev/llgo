package cl

import (
	"os"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/cabi"
	"github.com/xgo-dev/llgo/internal/optlevel"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

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
type T int8
//llgo:link (*T).Check C.check
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

func TestCVariadicValueReceiverWrapper(t *testing.T) {
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
	fn := mustNamedFunction(t, pkg.Module(), "foo.(*Wrapper).Check")
	if text := fn.String(); !strings.Contains(text, "call i32 (i32, ...) @check(") || strings.Contains(text, "musttail") {
		t.Fatalf("incompatible receiver ABI must retain a regular C call:\n%s", text)
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestCVariadicIncompatibleSignature(t *testing.T) {
	// Ordinary methods have no ellipsis to forward. Aggregate signatures need
	// instructions or ABI rewrites that cannot retain a musttail call followed
	// immediately by its return. Keep the existing lowering for both cases.
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
				t.Fatalf("incompatible wrapper must retain a regular C call:\n%s", text)
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
