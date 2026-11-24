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

func TestIntrinsicWrapperPerFunctionSetup(t *testing.T) {
	const src = `package wrapsetup
import _ "unsafe"

//llgo:link ctpop llvm.ctpop
func ctpop(x uint32) uint32

var CtpopVal = ctpop
`
	oldDisableInline := disableInline
	disableInline = true
	defer func() { disableInline = oldDisableInline }()

	ssapkg, _, files := buildGoSSAPkg(t, src)
	prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	prog.EnableFuncInfoMetadata(true)

	pkg, _, err := NewPackageExWithEmbedMetaOptions(prog, nil, nil, nil, ssapkg, files, nil, false, Options{Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	mod := pkg.Module()
	defer mod.Dispose()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("%v\n%s", err, mod.String())
	}
	ir := mod.String()
	fn := mustNamedFunction(t, mod, "wrapsetup.ctpop")
	if !strings.Contains(fn.String(), "noinline") {
		t.Fatalf("expected noinline attribute when disableInline=true:\n%s", fn.String())
	}
	if !strings.Contains(fn.String(), "!dbg") {
		t.Fatalf("expected debug metadata scope on wrapper when Debug=true:\n%s", fn.String())
	}
	if !strings.Contains(ir, llssa.FuncInfoMetadataName) {
		t.Fatalf("expected func-info metadata when FuncInfoMetadataEnabled=true:\n%s", ir)
	}
}
