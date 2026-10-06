package ssa_test

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"github.com/xgo-dev/llvm"
)

func TestSetjmpLongjmpIRPaths(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	fn := pkg.NewFunc("f", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(1)
	jb := b.AllocaSigjmpBuf()
	zero := prog.IntVal(0, prog.CInt())
	one := prog.IntVal(1, prog.CInt())
	_ = b.Sigsetjmp(jb, zero)
	b.Siglongjmp(jb, one)
	b.Longjmp(jb, one)
	b.Return()
	b.EndBuild()

	ir := pkg.Module().String()
	if !strings.Contains(ir, "setjmp") {
		t.Fatalf("expected setjmp/sigsetjmp symbol in IR, got:\n%s", ir)
	}
	if !strings.Contains(ir, "longjmp") {
		t.Fatalf("expected longjmp/siglongjmp symbol in IR, got:\n%s", ir)
	}
}

func TestSigjmpUsesSetjmpOnExplicitTarget(t *testing.T) {
	// The esp32 target resolves to the Linux/ARM Go compatibility surface.
	// Specify the resolved target up front instead of inheriting the host OS,
	// which would incorrectly make this test exercise the Windows UCRT ABI on
	// a Windows runner.
	prog := ssatest.NewProgram(t, &ssa.Target{
		GOOS: "linux", GOARCH: "arm", Target: "esp32",
	})
	pkg := prog.NewPackage("foo", "foo")

	fn := pkg.NewFunc("f", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(1)
	jb := b.AllocaSigjmpBuf()
	zero := prog.IntVal(0, prog.CInt())
	one := prog.IntVal(1, prog.CInt())
	_ = b.Sigsetjmp(jb, zero)
	b.Siglongjmp(jb, one)
	b.Return()
	b.EndBuild()

	ir := pkg.Module().String()
	if !strings.Contains(ir, "@setjmp") || !strings.Contains(ir, "@longjmp") {
		t.Fatalf("expected setjmp/longjmp fallback on explicit target, got:\n%s", ir)
	}
}

func TestWindowsSigjmpBufferAlignment(t *testing.T) {
	for _, arch := range []string{"386", "amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			prog := ssatest.NewProgram(t, &ssa.Target{GOOS: "windows", GOARCH: arch})
			pkg := prog.NewPackage("foo", "foo")

			fn := pkg.NewFunc("f", ssa.NoArgsNoRet, ssa.InGo)
			b := fn.MakeBody(1)
			_ = b.AllocaSigjmpBuf()
			b.Return()
			b.EndBuild()

			ir := pkg.Module().String()
			if !strings.Contains(ir, "alloca i8") || !strings.Contains(ir, "align 16") {
				t.Fatalf("expected a 16-byte-aligned Windows/%s jmp_buf allocation, got:\n%s", arch, ir)
			}
		})
	}
}

func TestWasmLateDeferBufferUsesFixedFrame(t *testing.T) {
	prog := ssatest.NewProgram(t, &ssa.Target{GOOS: "js", GOARCH: "wasm"})
	pkg := prog.NewPackage("foo", "foo")
	fn := pkg.NewFunc("f", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(2)
	b.Jump(fn.Block(1))
	b.SetBlock(fn.Block(1))
	jb := b.AllocaSigjmpBuf()
	b.Sigsetjmp(jb, prog.IntVal(0, prog.CInt()))
	b.Return()
	b.EndBuild()

	// A buffer reserved after the entry block is a dynamic shadow-stack
	// allocation. A caught native exception can discard it while the defer
	// frame still points at it, and subsequent calls overwrite the saved jump.
	entry := pkg.Module().NamedFunction("f").EntryBasicBlock()
	allocations := 0
	for block := entry; !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if !inst.IsAAllocaInst().IsNil() {
				allocations++
				if block != entry {
					t.Fatalf("jmp_buf is not in the fixed function frame:\n%s", pkg.Module().String())
				}
			}
		}
	}
	if allocations != 1 {
		t.Fatalf("got %d jump-buffer allocations, want 1", allocations)
	}
}

func TestWindowsSetjmpABI(t *testing.T) {
	tests := []struct {
		name       string
		arch       string
		llvmTarget string
		setjmp     string
		longjmp    string
	}{
		{name: "386", arch: "386", setjmp: "@_setjmp3", longjmp: "@longjmp"},
		{name: "amd64-msvc", arch: "amd64", setjmp: "@_setjmpex", longjmp: "@llgo_longjmp"},
		{name: "amd64-gnu", arch: "amd64", llvmTarget: "x86_64-w64-windows-gnu", setjmp: "@__intrinsic_setjmpex", longjmp: "@llgo_longjmp"},
		{name: "arm64", arch: "arm64", setjmp: "@llgo_setjmp", longjmp: "@llgo_longjmp"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prog := ssatest.NewProgram(t, &ssa.Target{GOOS: "windows", GOARCH: test.arch, LLVMTarget: test.llvmTarget})
			pkg := prog.NewPackage("foo", "foo")

			fn := pkg.NewFunc("f", ssa.NoArgsNoRet, ssa.InGo)
			b := fn.MakeBody(1)
			jb := b.AllocaSigjmpBuf()
			zero := prog.IntVal(0, prog.CInt())
			one := prog.IntVal(1, prog.CInt())
			_ = b.Sigsetjmp(jb, zero)
			b.Siglongjmp(jb, one)
			b.Return()
			b.EndBuild()

			ir := pkg.Module().String()
			if !strings.Contains(ir, test.setjmp) {
				t.Fatalf("Windows/%s IR does not call %s:\n%s", test.arch, test.setjmp, ir)
			}
			if !strings.Contains(ir, "returns_twice") {
				t.Fatalf("Windows/%s setjmp declaration is not marked returns_twice:\n%s", test.arch, ir)
			}
			if !strings.Contains(ir, test.longjmp) {
				t.Fatalf("Windows/%s IR does not call %s:\n%s", test.arch, test.longjmp, ir)
			}
			if test.arch == "amd64" && !strings.Contains(ir, "ptr null") {
				t.Fatalf("Windows/amd64 setjmp does not disable UCRT unwinding:\n%s", ir)
			}
			if (test.arch == "amd64" || test.arch == "arm64") && (strings.Contains(ir, "@llvm.frameaddress") || strings.Contains(ir, "@llvm.sponentry")) {
				t.Fatalf("Windows/%s IR unexpectedly captures a UCRT unwind frame:\n%s", test.arch, ir)
			}
			if strings.Contains(ir, "sigsetjmp") || strings.Contains(ir, "siglongjmp") {
				t.Fatalf("Windows/%s IR unexpectedly uses POSIX sigjmp symbols:\n%s", test.arch, ir)
			}
		})
	}
}

func TestWindowsDirectSetjmpABI(t *testing.T) {
	for _, test := range []struct {
		name       string
		arch       string
		llvmTarget string
		setjmp     string
		longjmp    string
	}{
		{name: "386", arch: "386", setjmp: "@_setjmp3", longjmp: "@longjmp"},
		{name: "amd64-msvc", arch: "amd64", setjmp: "@_setjmpex", longjmp: "@llgo_longjmp"},
		{name: "amd64-gnu", arch: "amd64", llvmTarget: "x86_64-w64-windows-gnu", setjmp: "@__intrinsic_setjmpex", longjmp: "@llgo_longjmp"},
		{name: "arm64", arch: "arm64", setjmp: "@llgo_setjmp", longjmp: "@llgo_longjmp"},
	} {
		t.Run(test.name, func(t *testing.T) {
			prog := ssatest.NewProgram(t, &ssa.Target{GOOS: "windows", GOARCH: test.arch, LLVMTarget: test.llvmTarget})
			pkg := prog.NewPackage("foo", "foo")

			fn := pkg.NewFunc("f", ssa.NoArgsNoRet, ssa.InGo)
			b := fn.MakeBody(1)
			jb := b.AllocaSigjmpBuf()
			one := prog.IntVal(1, prog.CInt())
			_ = b.Setjmp(jb)
			b.Longjmp(jb, one)
			b.Return()
			b.EndBuild()

			ir := pkg.Module().String()
			if !strings.Contains(ir, test.setjmp) || !strings.Contains(ir, test.longjmp) {
				t.Fatalf("Windows/%s direct setjmp ABI mismatch:\n%s", test.arch, ir)
			}
			if strings.Contains(ir, "sigsetjmp") || strings.Contains(ir, "siglongjmp") {
				t.Fatalf("Windows/%s direct setjmp unexpectedly uses POSIX signal variants:\n%s", test.arch, ir)
			}
		})
	}
}

func TestWindowsSetjmpRejectsUnsupportedArchitecture(t *testing.T) {
	// The official Windows LLVM package does not include the Mips backend.
	// RISC-V is present in every LLVM 22 payload used by LLGo and remains an
	// unsupported Windows setjmp architecture, which is the contract under test.
	prog := ssatest.NewProgram(t, &ssa.Target{
		GOOS: "windows", GOARCH: "riscv64", LLVMTarget: "riscv64-unknown-unknown",
	})
	pkg := prog.NewPackage("foo", "foo")
	fn := pkg.NewFunc("f", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(1)
	jb := b.AllocaSigjmpBuf()
	zero := prog.IntVal(0, prog.CInt())
	defer func() {
		got := recover()
		msg, ok := got.(string)
		if !ok || !strings.Contains(msg, "unsupported Windows architecture") {
			t.Fatalf("Sigsetjmp panic = %v, want unsupported-architecture diagnostic", got)
		}
	}()
	_ = b.Sigsetjmp(jb, zero)
}

func TestDeferInLoopContiguousDrainerGeneration(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	c1 := pkg.NewFunc("c1", ssa.NoArgsNoRet, ssa.InGo)
	b1 := c1.MakeBody(1)
	b1.Return()
	b1.EndBuild()

	c2 := pkg.NewFunc("c2", ssa.NoArgsNoRet, ssa.InGo)
	b2 := c2.MakeBody(1)
	b2.Return()
	b2.EndBuild()

	fn := pkg.NewFunc("main", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(1)
	fn.SetRecover(fn.MakeBlock())
	b.Return()
	b.SetBlockEx(fn.Block(0), ssa.BeforeLast, true)

	// Two contiguous loop defers should share one drain-loop generation pass.
	b.Defer(ssa.DeferInLoop, c1.Expr, ssa.Builder.Call)
	b.Defer(ssa.DeferInLoop, c2.Expr, ssa.Builder.Call)
	// Non-loop defer resets loop drainer state while walking deferred stmts.
	b.Defer(ssa.DeferAlways, c1.Expr, ssa.Builder.Call)
	b.EndBuild()

	ir := pkg.Module().String()
	if !strings.Contains(ir, "FreeDeferNode") {
		t.Fatalf("expected defer node drain/free in IR, got:\n%s", ir)
	}
}
