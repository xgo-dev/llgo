package llvmfmv

import (
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestVariantCOMDAT(t *testing.T) {
	for _, selection := range []string{"any", "samesize"} {
		t.Run(selection, func(t *testing.T) {
			mod := parseModule(t, `
target triple = "x86_64-pc-windows-msvc"
$generic = comdat `+selection+`
declare i1 @avx2() "llgo.cpu.query"="x86.avx2"
declare float @external(float) "llgo.fmv.avx2-entry"
define linkonce float @generic(float %x) comdat {
 %enabled = call i1 @avx2()
 br i1 %enabled, label %fast, label %slow
fast:
 %y = fadd float %x, %x
 ret float %y
slow:
 ret float %x
}
define float @plain(float %x) "llgo.fmv.avx2-entry" {
 ret float %x
}
`)
			original := mod.NamedFunction("generic").Comdat()
			if err := Run(mod, "v1"); err != nil {
				t.Fatal(err)
			}
			variant := mod.NamedFunction("generic" + variantSuffix)
			cd := variant.Comdat()
			if cd.C == nil || cd.C == original.C || cd.SelectionKind() != original.SelectionKind() ||
				cd.C != mod.Comdat(variant.Name()).C {
				t.Fatalf("variant must have its own COMDAT with the original selection kind:\n%s", mod.String())
			}
			for _, name := range []string{"generic" + baselineSuffix, "generic" + resolverSuffix, "plain" + variantSuffix, "external" + variantSuffix} {
				if mod.NamedFunction(name).Comdat().C != nil {
					t.Fatalf("%s gained an unnecessary COMDAT", name)
				}
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
