package plan9asm

import (
	"path/filepath"
	"runtime"
	"testing"

	extplan9asm "github.com/xgo-dev/plan9asm"
)

func TestSigsForStdlibInternalBytealg386Helpers(t *testing.T) {
	goroot := runtime.GOROOT()
	if goroot == "" {
		t.Skip("GOROOT not available")
	}
	pkg := loadStdlibInternalBytealgForTarget(t, "windows", "386")

	tests := map[string]map[string]extplan9asm.FuncSig{
		filepath.Join(goroot, "src", "internal", "bytealg", "compare_386.s"): {
			"internal/bytealg.cmpbody": {
				Args:    []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.I32, extplan9asm.Ptr, extplan9asm.I32, extplan9asm.Ptr},
				Ret:     extplan9asm.Void,
				ArgRegs: []extplan9asm.Reg{extplan9asm.SI, extplan9asm.BX, extplan9asm.DI, extplan9asm.DX, extplan9asm.AX},
			},
		},
		filepath.Join(goroot, "src", "internal", "bytealg", "equal_386.s"): {
			"internal/bytealg.memeqbody": {
				Args:    []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.Ptr, extplan9asm.I32, extplan9asm.Ptr},
				Ret:     extplan9asm.Void,
				ArgRegs: []extplan9asm.Reg{extplan9asm.SI, extplan9asm.DI, extplan9asm.BX, extplan9asm.AX},
			},
		},
	}

	for path, wantSigs := range tests {
		tr, err := TranslateFileForPkg(pkg, path, "windows", "386", nil)
		if err != nil {
			t.Fatalf("translate %s: %v", path, err)
		}
		for name, want := range wantSigs {
			got, ok := tr.Signatures[name]
			if !ok {
				t.Fatalf("missing symbol %s in %s", name, path)
			}
			if err := checkSig(got, want); err != nil {
				t.Fatalf("%s (%s): %v", name, path, err)
			}
		}
	}
}
