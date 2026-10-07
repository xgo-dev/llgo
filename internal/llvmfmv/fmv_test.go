package llvmfmv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func parseModule(t *testing.T, source string) llvm.Module {
	t.Helper()
	ctx := llvm.NewContext()
	t.Cleanup(ctx.Dispose)
	path := filepath.Join(t.TempDir(), "fmv.ll")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	buf, err := llvm.NewMemoryBufferFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mod, err := ctx.ParseIR(buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mod.Dispose)
	return mod
}

const fixture = `
target triple = "x86_64-unknown-linux-gnu"
@address = global ptr @root
declare i1 @avx2() "llgo.cpu.query"="x86.avx2"
declare i1 @fma() "llgo.cpu.query"="x86.fma"
declare <4 x float> @external(<4 x float>) "llgo.fmv.avx2-entry"
declare <4 x float> @assembly(<4 x float>)
declare void @consume(ptr)
define void @addressOnly() {
  call void @consume(ptr @avx2)
  ret void
}
define <4 x float> @helper(<4 x float> %x) "llgo.fmv.avx2-entry" {
  %y = call <4 x float> @external(<4 x float> %x)
  ret <4 x float> %y
}
define <4 x float> @root(<4 x float> %x, ptr %indirect) "disable-tail-calls"="true" {
  %ok = call i1 @avx2()
  %no = xor i1 %ok, true
  br i1 %no, label %fallback, label %fast
fast:
  call void asm sideeffect "__llgo_pcsite_000000000000002a_$\7B:uid\7D:\0A.quad 0x000000000000002a", ""()
  %y = call <4 x float> @helper(<4 x float> %x)
  %z = call <4 x float> %indirect(<4 x float> %y)
  %a = call <4 x float> @assembly(<4 x float> %z)
  %f = call i1 @fma()
  br i1 %f, label %hasfma, label %nofma
hasfma:
  ret <4 x float> %a
nofma:
  ret <4 x float> %z
fallback:
  ret <4 x float> zeroinitializer
}
define <4 x float> @recursive(<4 x float> %x, i32 %n) "llgo.fmv.avx2-entry" {
  %end = icmp eq i32 %n, 0
  br i1 %end, label %done, label %again
again:
  %next = sub i32 %n, 1
  %r = call <4 x float> @recursive(<4 x float> %x, i32 %next)
  ret <4 x float> %r
done:
  ret <4 x float> %x
}
define <8 x float> @wide(<8 x float> %x) "llgo.fmv.avx2-entry" {
  %ok = call i1 @avx2()
  ret <8 x float> %x
}
!llgo.funcinfo = !{!0}
!llgo.pcline = !{!1}
!0 = !{i32 1, !"root", !"example.root", !"example.go", i32 10, i32 1}
!1 = !{i32 1, i64 42, !"root", !"example.go", i32 12, i32 1}
`

func TestEarlySpecialization(t *testing.T) {
	mod := parseModule(t, fixture)
	if err := Run(mod); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("%v\n%s", err, mod.String())
	}
	root := mod.NamedFunction("root").String()
	variant := mod.NamedFunction("root.__llgo_fmv_avx2").String()
	if !strings.Contains(mod.NamedGlobal("address").String(), "ptr @root") || strings.Contains(mod.NamedGlobal("address").String(), "__llgo_fmv") {
		t.Fatal("function address no longer names baseline entry")
	}
	for _, want := range []string{"call i1 @avx2()", "musttail call <4 x float> @root.__llgo_fmv_avx2", "fallback:"} {
		if !strings.Contains(root, want) {
			t.Errorf("baseline missing %q:\n%s", want, root)
		}
	}
	for _, want := range []string{"@helper.__llgo_fmv_avx2(", "@fma()", "%indirect(", "@assembly("} {
		if !strings.Contains(variant, want) {
			t.Errorf("variant missing %q:\n%s", want, variant)
		}
	}
	for _, unwanted := range []string{"@avx2()", "fallback:", "000000000000002a"} {
		if strings.Contains(variant, unwanted) {
			t.Errorf("variant retains %q:\n%s", unwanted, variant)
		}
	}
	if !strings.Contains(mod.NamedFunction("helper.__llgo_fmv_avx2").String(), "@external.__llgo_fmv_avx2(") ||
		!strings.Contains(mod.NamedFunction("recursive.__llgo_fmv_avx2").String(), "@recursive.__llgo_fmv_avx2(") {
		t.Fatal("direct call propagation missing")
	}
	if !mod.NamedFunction("wide.__llgo_fmv_avx2").IsNil() || !mod.NamedFunction("assembly.__llgo_fmv_avx2").IsNil() {
		t.Fatal("unsupported ABI or unpromised external entry specialized")
	}
	if !mod.NamedFunction("addressOnly.__llgo_fmv_avx2").IsNil() {
		t.Fatal("taking a query's address must not establish a CPU guard")
	}
	for _, want := range []string{`!"root.__llgo_fmv_avx2", !"example.root"`, `!"root.__llgo_fmv_avx2", !"example.go"`, `"target-features"="+avx,+avx2"`} {
		if !strings.Contains(mod.String(), want) {
			t.Errorf("missing %s", want)
		}
	}
	before := mod.String()
	if err := Run(mod); err != nil {
		t.Fatal(err)
	}
	if mod.String() != before {
		t.Fatal("FMV is not idempotent")
	}
}

func TestOtherTargetUnchanged(t *testing.T) {
	mod := parseModule(t, strings.Replace(fixture, "x86_64-unknown-linux-gnu", "aarch64-unknown-linux-gnu", 1))
	before := mod.String()
	if err := Run(mod); err != nil || mod.String() != before {
		t.Fatalf("non-amd64 module changed: %v", err)
	}
}

func TestSymbolCollision(t *testing.T) {
	mod := parseModule(t, fixture+"\ndeclare void @root.__llgo_fmv_avx2()\n")
	before := mod.String()
	if err := Run(mod); err == nil || !strings.Contains(err.Error(), "symbol collision") {
		t.Fatalf("expected collision diagnostic, got %v", err)
	}
	if mod.String() != before {
		t.Fatal("collision partially mutated module")
	}
}
