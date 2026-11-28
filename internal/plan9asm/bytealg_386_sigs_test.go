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
		t.Fatal("GOROOT not available")
	}
	pkg := loadStdlibInternalBytealgForTarget(t, "windows", "386")

	tests := map[string]map[string]extplan9asm.FuncSig{
		filepath.Join(goroot, "src", "internal", "bytealg", "compare_386.s"): {
			"internal/bytealg.Compare": {
				Args: []extplan9asm.LLVMType{"{ ptr, i32, i32 }", "{ ptr, i32, i32 }"},
				Ret:  extplan9asm.I32,
			},
			"runtime.cmpstring": {
				Args: []extplan9asm.LLVMType{"{ ptr, i32 }", "{ ptr, i32 }"},
				Ret:  extplan9asm.I32,
			},
		},
		filepath.Join(goroot, "src", "internal", "bytealg", "equal_386.s"): {
			"runtime.memequal": {
				Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.Ptr, extplan9asm.I32},
				Ret:  extplan9asm.I1,
			},
			"runtime.memequal_varlen": {
				Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.Ptr},
				Ret:  extplan9asm.I1,
			},
		},
	}

	for path, wantSigs := range tests {
		t.Run(filepath.Base(path), func(t *testing.T) {
			requireBytealgOriginalFileContract(t, pkg, path, "windows", "386", wantSigs)
		})
	}
}
