package ssa_test

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
)

func TestSetjmpLongjmpIRPaths(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	fn := pkg.NewFunc("f", ssa.NoArgsNoRet, ssa.GoDecl)
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

	fn := pkg.NewFunc("f", ssa.NoArgsNoRet, ssa.GoDecl)
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

			fn := pkg.NewFunc("f", ssa.NoArgsNoRet, ssa.GoDecl)
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

			fn := pkg.NewFunc("f", ssa.NoArgsNoRet, ssa.GoDecl)
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

			fn := pkg.NewFunc("f", ssa.NoArgsNoRet, ssa.GoDecl)
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
	fn := pkg.NewFunc("f", ssa.NoArgsNoRet, ssa.GoDecl)
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

	c1 := pkg.NewFunc("c1", ssa.NoArgsNoRet, ssa.GoDecl)
	b1 := c1.MakeBody(1)
	b1.Return()
	b1.EndBuild()

	c2 := pkg.NewFunc("c2", ssa.NoArgsNoRet, ssa.GoDecl)
	b2 := c2.MakeBody(1)
	b2.Return()
	b2.EndBuild()

	fn := pkg.NewFunc("main", ssa.NoArgsNoRet, ssa.GoDecl)
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

func TestDarwinArm64DeferSetjmpABI(t *testing.T) {
	prog := ssatest.NewProgram(t, &ssa.Target{GOOS: "darwin", GOARCH: "arm64"})
	pkg := prog.NewPackage("foo", "foo")

	cleanup := pkg.NewFunc("cleanup", ssa.NoArgsNoRet, ssa.GoDecl)
	cb := cleanup.MakeBody(1)
	cb.Return()
	cb.EndBuild()

	// Defer frames on darwin/arm64 use 16-byte-aligned buffers and llgo_setjmp
	// (marked returns_twice), while direct Sigsetjmp/Siglongjmp calls keep
	// using libc sigsetjmp/siglongjmp because their buffer layouts differ.
	deferFn := pkg.NewFunc("withDefer", ssa.NoArgsNoRet, ssa.GoDecl)
	db := deferFn.MakeBody(1)
	deferFn.SetRecover(deferFn.MakeBlock())
	db.Defer(ssa.DeferAlways, cleanup.Expr, ssa.Builder.Call)
	db.RunDefers()
	db.Return()
	db.EndBuild()

	libcFn := pkg.NewFunc("directSigjmp", ssa.NoArgsNoRet, ssa.GoDecl)
	lb := libcFn.MakeBody(1)
	jb := lb.AllocaSigjmpBuf()
	zero := prog.IntVal(0, prog.CInt())
	one := prog.IntVal(1, prog.CInt())
	_ = lb.Sigsetjmp(jb, zero)
	lb.Siglongjmp(jb, one)
	lb.Return()
	lb.EndBuild()

	modIR := pkg.Module().String()
	deferIR := pkg.Module().NamedFunction("withDefer").String()
	libcIR := pkg.Module().NamedFunction("directSigjmp").String()

	if !strings.Contains(deferIR, "alloca i8") || !strings.Contains(deferIR, "align 16") {
		t.Fatalf("darwin/arm64 defer jmpbuf missing 16-byte alignment:\n%s", deferIR)
	}
	if !strings.Contains(deferIR, "@llgo_setjmp") {
		t.Fatalf("darwin/arm64 defer did not lower to @llgo_setjmp:\n%s", deferIR)
	}
	if strings.Contains(deferIR, "@sigsetjmp") {
		t.Fatalf("darwin/arm64 defer unexpectedly called libc @sigsetjmp:\n%s", deferIR)
	}
	if !strings.Contains(modIR, "returns_twice") {
		t.Fatalf("darwin/arm64 @llgo_setjmp declaration missing returns_twice:\n%s", modIR)
	}
	if !strings.Contains(libcIR, ".Sigsetjmp") || !strings.Contains(libcIR, ".Siglongjmp") {
		t.Fatalf("darwin/arm64 direct Sigsetjmp/Siglongjmp did not preserve runtime libc ABI:\n%s", libcIR)
	}
	if strings.Contains(libcIR, "@llgo_setjmp") || strings.Contains(libcIR, "@llgo_longjmp") {
		t.Fatalf("darwin/arm64 direct Sigsetjmp/Siglongjmp unexpectedly used defer ABI:\n%s", libcIR)
	}
}
