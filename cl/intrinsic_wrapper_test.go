package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestIntrinsicFunctionValueWrappers(t *testing.T) {
	const src = `package wrapcase
import _ "unsafe"

//llgo:link ctpop llvm.ctpop
func ctpop(x uint32) uint32 { panic("ignored") }

//llgo:link umul llvm.umul.with.overflow
func umul(a, b uint64) (uint64, bool) { panic("ignored") }

//llgo:link keepAlive llgo.keepAlive
func keepAlive(v any)

//llgo:link unreachable llgo.unreachable
func unreachable()

//llgo:link umulInstr llgo.umulOverflow
func umulInstr(a, b uint64) (uint64, bool)

var (
	CtpopVal       = ctpop
	UMulVal        = umul
	KeepAliveVal   = keepAlive
	UnreachableVal = unreachable
	UMulInstrVal   = umulInstr
)
`
	ssapkg, _, files := buildGoSSAPkg(t, src)
	prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	pkg, err := NewPackage(prog, ssapkg, files)
	if err != nil {
		t.Fatal(err)
	}
	mod := pkg.Module()
	defer mod.Dispose()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("%v\n%s", err, mod.String())
	}
	for _, tc := range []struct {
		sym  string
		want string
	}{
		{"wrapcase.ctpop", "@llvm.ctpop.i32"},
		{"wrapcase.umul", "@llvm.umul.with.overflow.i64"},
		{"wrapcase.keepAlive", "ret void"},
		{"wrapcase.unreachable", "unreachable"},
		{"wrapcase.umulInstr", "@llvm.umul.with.overflow.i64"},
	} {
		fn := mustNamedFunction(t, mod, tc.sym)
		if fn.IsDeclaration() || fn.Linkage() != llvm.LinkOnceAnyLinkage {
			t.Fatalf("%s: expected linkonce wrapper definition, got:\n%s", tc.sym, fn.String())
		}
		if !strings.Contains(fn.String(), tc.want) {
			t.Fatalf("%s: wrapper IR missing %q:\n%s", tc.sym, tc.want, fn.String())
		}
	}
}
