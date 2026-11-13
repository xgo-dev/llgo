package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestAESIntrinsicLowering(t *testing.T) {
	ssaPkg, _, files := buildGoSSAPkg(t, `package foo
import "unsafe"
//go:linkname buffer llgo.aesAlloca
func buffer(uintptr) unsafe.Pointer
//go:linkname round llgo.aesRound
func round(int, *byte, *byte, *byte)
//go:linkname subword llgo.aesSubWord
func subword(uint32) uint32
func kernel(dst, src, key *byte, x uint32) uint32 {
	state := (*byte)(buffer(16))
	round(0, state, src, key)
	round(1, dst, state, key)
	return subword(x)
}
`)
	prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	pkg, err := NewPackage(prog, ssaPkg, files)
	if err != nil {
		t.Fatal(err)
	}
	mod := pkg.Module()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	ir := mod.String()
	for _, want := range []string{"alloca [1 x <16 x i8>], align 16", "llvm.x86.aesni.aesenc", "llvm.x86.aesni.aeskeygenassist"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing %q in AES lowering:\n%s", want, ir)
		}
	}
}

// Malformed private declarations must fail before reaching LLVM with invalid
// operand types, a dynamic stack buffer, or an unsupported instruction code.
func TestAESIntrinsicDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name, intrinsic, declaration, caller, arch, want string
	}{
		{"buffer arity", "aesAlloca", "func instr() unsafe.Pointer", "func bad() { _ = instr() }", "amd64", "aesAlloca: expected a constant buffer size"},
		{"buffer dynamic", "aesAlloca", "func instr(uintptr) unsafe.Pointer", "func bad(n uintptr) { _ = instr(n) }", "amd64", "aesAlloca: expected 16 or 128 bytes"},
		{"buffer size", "aesAlloca", "func instr(uintptr) unsafe.Pointer", "func bad() { _ = instr(32) }", "amd64", "aesAlloca: expected 16 or 128 bytes"},
		{"round arity", "aesRound", "func instr(int)", "func bad() { instr(0) }", "amd64", "aesRound: expected an operation and three pointers"},
		{"round dynamic", "aesRound", "func instr(int, *byte, *byte, *byte)", "func bad(n int, p *byte) { instr(n, p, p, p) }", "amd64", "aesRound: invalid constant operation"},
		{"round opcode", "aesRound", "func instr(int, *byte, *byte, *byte)", "func bad(p *byte) { instr(7, p, p, p) }", "amd64", "aesRound: invalid constant operation"},
		{"round pointers", "aesRound", "func instr(int, *uint32, *uint32, *uint32)", "func bad(p *uint32) { instr(0, p, p, p) }", "amd64", "aesRound: expected *byte operands"},
		{"subword arity", "aesSubWord", "func instr() uint32", "func bad() uint32 { return instr() }", "amd64", "aesSubWord: expected one uint32"},
		{"subword type", "aesSubWord", "func instr(uint64) uint64", "func bad(x uint64) uint64 { return instr(x) }", "amd64", "aesSubWord: expected uint32"},
		{"unsupported target", "aesSubWord", "func instr(uint32) uint32", "func bad(x uint32) uint32 { return instr(x) }", "wasm", "AES intrinsic on unsupported architecture"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "package foo\nimport \"unsafe\"\nvar _ unsafe.Pointer\n//go:linkname instr llgo." + tc.intrinsic + "\n" + tc.declaration + "\n" + tc.caller
			ssaPkg, _, files := buildGoSSAPkg(t, src)
			target := &llssa.Target{GOOS: "linux", GOARCH: tc.arch}
			if tc.arch == "wasm" {
				target.GOOS = "js"
			}
			prog := newLLSSAProgForTarget(t, target)
			defer prog.Dispose()
			mustPanicContains(t, tc.want, func() {
				if _, err := NewPackage(prog, ssaPkg, files); err != nil {
					panic(err)
				}
			})
		})
	}
}
