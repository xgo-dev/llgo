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
	prog := ssatest.NewProgram(t, nil)
	prog.Target().Target = "esp32"
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

func TestWindowsSetjmpABI(t *testing.T) {
	tests := []struct {
		arch    string
		setjmp  string
		longjmp string
	}{
		{arch: "386", setjmp: "@_setjmp3", longjmp: "@longjmp"},
		{arch: "amd64", setjmp: "@_setjmpex", longjmp: "@longjmp"},
		{arch: "arm64", setjmp: "@llgo_setjmp", longjmp: "@llgo_longjmp"},
	}
	for _, test := range tests {
		t.Run(test.arch, func(t *testing.T) {
			prog := ssatest.NewProgram(t, &ssa.Target{GOOS: "windows", GOARCH: test.arch})
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

func TestWindowsSetjmpRejectsUnsupportedArchitecture(t *testing.T) {
	prog := ssatest.NewProgram(t, &ssa.Target{GOOS: "windows", GOARCH: "mips"})
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
