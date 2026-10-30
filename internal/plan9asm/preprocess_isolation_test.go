package plan9asm

import "testing"

func TestTranslatePreprocessedSourceCannotBorrowAmbientMacros(t *testing.T) {
	t.Setenv("GOAMD64", "v4")
	pkg := mustTestPackage(t, "example.com/macros", "package macros\nconst Value = 42\nfunc Read() uint64\n")
	for _, operand := range []string{"GOAMD64_v4", "const_Value"} {
		t.Run(operand, func(t *testing.T) {
			// Neither name is defined by the selected v1 invocation. const_Value
			// exists only when this source actually includes generated go_asm.h.
			source := []byte("TEXT ·Read(SB),4,$0-8\nMOVQ $" + operand + ",AX\nMOVQ AX,ret+0(FP)\nRET\n")
			tr, err := TranslateSourceModuleForPkgWithOptions(pkg, "read.s", source, "linux", "amd64", TranslateOptions{GOAMD64: "v1"})
			if err == nil {
				tr.Module.Dispose()
				t.Fatal("undefined selected-source operand was silently defined by a second expansion")
			}
		})
	}
}
