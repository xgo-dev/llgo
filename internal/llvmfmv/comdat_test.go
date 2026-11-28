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
declare i1 @fma() "llgo.cpu.query"="x86.fma"
declare <4 x float> @llvm.fma.v4f32(<4 x float>, <4 x float>, <4 x float>)
declare float @external(float)
define linkonce <4 x float> @generic(<4 x float> %x) comdat {
 %enabled = call i1 @fma()
 br i1 %enabled, label %fast, label %slow
fast:
 %y = call <4 x float> @llvm.fma.v4f32(<4 x float> %x, <4 x float> %x, <4 x float> %x)
 ret <4 x float> %y
slow:
 ret <4 x float> %x
}
define float @plain(float %x) {
 ret float %x
}
`)
			original := mod.NamedFunction("generic").Comdat()
			if err := Run(mod, "v1"); err != nil {
				t.Fatal(err)
			}
			fn := mod.NamedFunction("generic")
			if fn.Linkage() != llvm.LinkOnceAnyLinkage || fn.Comdat().C != original.C {
				t.Fatal("public generic entry lost its COMDAT identity")
			}
			for _, suffix := range []string{symbolPrefix + "guard_fma", symbolPrefix + "guard_base", resolverSuffix} {
				impl := mod.NamedFunction("generic" + suffix)
				if impl.IsNil() || impl.Linkage() != llvm.InternalLinkage || impl.Comdat().C != nil {
					t.Fatalf("expected a module-local implementation of generic%s", suffix)
				}
			}
			if !mod.NamedFunction("plain"+symbolPrefix+"guard_fma").IsNil() ||
				!mod.NamedFunction("external"+symbolPrefix+"guard_fma").IsNil() {
				t.Fatal("unrelated definitions and external declarations must not grow variants")
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
