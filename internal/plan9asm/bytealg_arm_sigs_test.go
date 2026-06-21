//go:build !llgo
// +build !llgo

package plan9asm

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goplus/llgo/internal/packages"
	extplan9asm "github.com/xgo-dev/plan9asm"
)

func loadStdlibInternalBytealgForTarget(t *testing.T, goos, goarch string) *packages.Package {
	t.Helper()
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedTypesInfo | packages.NeedImports,
		Env:  append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch),
	}
	pkgs, err := packages.LoadEx(nil, nil, cfg, "internal/bytealg")
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].Types == nil {
		t.Fatalf("load internal/bytealg: got %d pkgs, types=%v", len(pkgs), pkgs[0].Types)
	}
	return pkgs[0]
}

func TestSigsForStdlibInternalBytealgArmHelpers(t *testing.T) {
	goroot := runtime.GOROOT()
	if goroot == "" {
		t.Skip("GOROOT not available")
	}
	pkg := loadStdlibInternalBytealgForTarget(t, "linux", "arm")

	tests := map[string]map[string]extplan9asm.FuncSig{
		filepath.Join(goroot, "src", "internal", "bytealg", "compare_arm.s"): {
			"internal/bytealg.cmpbody": {
				Args:    []extplan9asm.LLVMType{extplan9asm.Ptr, "i32", extplan9asm.Ptr, "i32", extplan9asm.Ptr},
				Ret:     extplan9asm.Void,
				ArgRegs: []extplan9asm.Reg{"R2", "R0", "R3", "R1", "R7"},
			},
		},
		filepath.Join(goroot, "src", "internal", "bytealg", "count_arm.s"): {
			"internal/bytealg.countbytebody": {
				Args:    []extplan9asm.LLVMType{extplan9asm.Ptr, "i32", "i8", extplan9asm.Ptr},
				Ret:     extplan9asm.Void,
				ArgRegs: []extplan9asm.Reg{"R0", "R1", "R2", "R7"},
			},
		},
		filepath.Join(goroot, "src", "internal", "bytealg", "equal_arm.s"): {
			"internal/bytealg.memeqbody": {
				Args:    []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.Ptr, "i32", extplan9asm.Ptr},
				Ret:     extplan9asm.Void,
				ArgRegs: []extplan9asm.Reg{"R0", "R2", "R1", "R7"},
			},
		},
		filepath.Join(goroot, "src", "internal", "bytealg", "indexbyte_arm.s"): {
			"internal/bytealg.indexbytebody": {
				Args:    []extplan9asm.LLVMType{extplan9asm.Ptr, "i32", "i8", extplan9asm.Ptr},
				Ret:     extplan9asm.Void,
				ArgRegs: []extplan9asm.Reg{"R0", "R1", "R2", "R5"},
			},
		},
	}

	for path, wantSigs := range tests {
		tr, err := TranslateFileForPkg(pkg, path, "linux", "arm", nil)
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
