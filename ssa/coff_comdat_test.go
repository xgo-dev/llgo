package ssa

import (
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestWindowsODRDefinitionsUseCOMDAT(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "windows", GOARCH: "amd64"})
	defer prog.Dispose()
	pkg := prog.NewPackage("example.com/p", "example.com/p")

	global := pkg.NewVar("shared.global", types.NewPointer(types.Typ[types.Int]), InGo)
	global.Init(prog.IntVal(0, prog.Int()))
	pkg.setODRLinkage(global.impl, llvm.WeakODRLinkage)
	fn := pkg.NewFuncEx("shared.func", NoArgsNoRet, InGo, false, true)
	body := fn.MakeBody(1)
	body.Return()
	body.EndBuild()

	ir := pkg.String()
	for _, want := range []string{
		`$shared.global = comdat any`,
		`$shared.func = comdat any`,
		`@shared.global = weak_odr global i64 0, comdat`,
		`define linkonce void @shared.func() #0 comdat`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("Windows ODR IR does not contain %q:\n%s", want, ir)
		}
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("Windows ODR module is invalid: %v\n%s", err, ir)
	}
}

func TestUnixODRDefinitionsDoNotGainCOMDAT(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	pkg := prog.NewPackage("example.com/p", "example.com/p")

	global := pkg.NewVar("shared.global", types.NewPointer(types.Typ[types.Int]), InGo)
	global.Init(prog.IntVal(0, prog.Int()))
	pkg.setODRLinkage(global.impl, llvm.WeakODRLinkage)

	if ir := pkg.String(); strings.Contains(ir, "comdat") {
		t.Fatalf("non-Windows ODR IR unexpectedly contains COMDAT:\n%s", ir)
	}
}
