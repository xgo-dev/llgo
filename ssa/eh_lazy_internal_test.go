package ssa

import (
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestIsTerminatorInst(t *testing.T) {
	if isTerminatorInst(llvm.Value{}) {
		t.Fatal("isTerminatorInst(nil) = true, want false")
	}
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	pkg := prog.NewPackage("eh", "example.com/eh")
	fn := pkg.NewFunc("f", NoArgsNoRet, GoDecl)
	b := fn.MakeBody(1)
	slot := b.Alloca(prog.IntVal(8, prog.Uintptr()))
	if isTerminatorInst(slot.impl) {
		t.Fatal("isTerminatorInst(alloca) = true, want false")
	}
	b.Return()
	if !isTerminatorInst(fn.Block(0).last.LastInstruction()) {
		t.Fatal("isTerminatorInst(ret) = false, want true")
	}
}
