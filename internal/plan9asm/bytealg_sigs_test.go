//go:build !llgo
// +build !llgo

package plan9asm

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/packages"
	extplan9asm "github.com/xgo-dev/plan9asm"
)

func TestSigsForStdlibInternalBytealgArm64(t *testing.T) {
	if runtime.GOARCH != "arm64" {
		t.Skip("host is not arm64")
	}
	goroot := runtime.GOROOT()
	if goroot == "" {
		t.Skip("GOROOT not available")
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedTypesInfo | packages.NeedImports,
		Env:  os.Environ(),
	}
	pkgs, err := packages.LoadEx(nil, nil, cfg, "internal/bytealg")
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].Types == nil {
		t.Fatalf("load internal/bytealg: got %d pkgs, types=%v", len(pkgs), pkgs[0].Types)
	}
	pkg := pkgs[0]

	sfiles := map[string]map[string]extplan9asm.FuncSig{
		filepath.Join(goroot, "src", "internal", "bytealg", "compare_arm64.s"): {
			"internal/bytealg.Compare": {
				Args: []extplan9asm.LLVMType{"{ ptr, i64, i64 }", "{ ptr, i64, i64 }"},
				Ret:  extplan9asm.I64,
			},
			"runtime.cmpstring": {
				Args: []extplan9asm.LLVMType{"{ ptr, i64 }", "{ ptr, i64 }"},
				Ret:  extplan9asm.I64,
			},
			"internal/bytealg.cmpbody": {
				Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.I64, extplan9asm.Ptr, extplan9asm.I64},
				Ret:  extplan9asm.I64,
			},
		},
		filepath.Join(goroot, "src", "internal", "bytealg", "count_arm64.s"): {
			"internal/bytealg.Count": {
				Args: []extplan9asm.LLVMType{"{ ptr, i64, i64 }", "i8"},
				Ret:  extplan9asm.I64,
			},
			"internal/bytealg.CountString": {
				Args: []extplan9asm.LLVMType{"{ ptr, i64 }", "i8"},
				Ret:  extplan9asm.I64,
			},
		},
		filepath.Join(goroot, "src", "internal", "bytealg", "index_arm64.s"): {
			"internal/bytealg.Index": {
				Args: []extplan9asm.LLVMType{"{ ptr, i64, i64 }", "{ ptr, i64, i64 }"},
				Ret:  extplan9asm.I64,
			},
			"internal/bytealg.IndexString": {
				Args: []extplan9asm.LLVMType{"{ ptr, i64 }", "{ ptr, i64 }"},
				Ret:  extplan9asm.I64,
			},
		},
		filepath.Join(goroot, "src", "internal", "bytealg", "indexbyte_arm64.s"): {
			"internal/bytealg.IndexByte": {
				Args: []extplan9asm.LLVMType{"{ ptr, i64, i64 }", "i8"},
				Ret:  extplan9asm.I64,
			},
			"internal/bytealg.IndexByteString": {
				Args: []extplan9asm.LLVMType{"{ ptr, i64 }", "i8"},
				Ret:  extplan9asm.I64,
			},
		},
		filepath.Join(goroot, "src", "internal", "bytealg", "equal_arm64.s"): {
			"runtime.memequal_varlen": {
				Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.Ptr},
				Ret:  "i1",
			},
			"runtime.memequal": {
				Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.Ptr, extplan9asm.I64},
				Ret:  "i1",
			},
		},
	}
	// Go 1.21 arm64 equal_arm64.s contains an extra helper TEXT memeqbody<>.
	// Newer toolchains inline it into runtime·memequal and drop this symbol.
	if src, err := os.ReadFile(filepath.Join(goroot, "src", "internal", "bytealg", "equal_arm64.s")); err == nil {
		if string(src) != "" && containsTextSymbol(string(src), "memeqbody<>") {
			sfiles[filepath.Join(goroot, "src", "internal", "bytealg", "equal_arm64.s")]["internal/bytealg.memeqbody"] = extplan9asm.FuncSig{
				Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.Ptr, extplan9asm.I64},
				Ret:  extplan9asm.I1,
			}
		}
	}
	// Go toolchains may keep or drop indexbody<> in index_arm64.s.
	if src, err := os.ReadFile(filepath.Join(goroot, "src", "internal", "bytealg", "index_arm64.s")); err == nil {
		if string(src) != "" && containsTextSymbol(string(src), "indexbody<>") {
			sfiles[filepath.Join(goroot, "src", "internal", "bytealg", "index_arm64.s")]["internal/bytealg.indexbody"] = extplan9asm.FuncSig{
				Args:    []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.I64, extplan9asm.Ptr, extplan9asm.I64, extplan9asm.Ptr},
				Ret:     extplan9asm.Void,
				ArgRegs: []extplan9asm.Reg{"R0", "R1", "R2", "R3", "R9"},
			}
		}
	}
	// Go toolchains may keep or drop indexbytebody<> in indexbyte_arm64.s.
	if src, err := os.ReadFile(filepath.Join(goroot, "src", "internal", "bytealg", "indexbyte_arm64.s")); err == nil {
		if string(src) != "" && containsTextSymbol(string(src), "indexbytebody<>") {
			sfiles[filepath.Join(goroot, "src", "internal", "bytealg", "indexbyte_arm64.s")]["internal/bytealg.indexbytebody"] = extplan9asm.FuncSig{
				Args:    []extplan9asm.LLVMType{extplan9asm.Ptr, "i8", extplan9asm.I64, extplan9asm.Ptr},
				Ret:     extplan9asm.Void,
				ArgRegs: []extplan9asm.Reg{"R0", "R1", "R2", "R8"},
			}
		}
	}
	// Go toolchains may keep or drop countbytebody<> in count_arm64.s.
	if src, err := os.ReadFile(filepath.Join(goroot, "src", "internal", "bytealg", "count_arm64.s")); err == nil {
		if string(src) != "" && containsTextSymbol(string(src), "countbytebody<>") {
			sfiles[filepath.Join(goroot, "src", "internal", "bytealg", "count_arm64.s")]["internal/bytealg.countbytebody"] = extplan9asm.FuncSig{
				Args:    []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.I64, "i8", extplan9asm.Ptr},
				Ret:     extplan9asm.Void,
				ArgRegs: []extplan9asm.Reg{"R0", "R2", "R1", "R8"},
			}
		}
	}

	for path, wantSigs := range sfiles {
		tr, err := TranslateFileForPkg(pkg, path, runtime.GOOS, "arm64", nil)
		if err != nil {
			t.Fatalf("translate %s: %v", path, err)
		}
		if len(tr.Functions) != len(wantSigs) {
			t.Fatalf("func count %s: got %d, want %d", path, len(tr.Functions), len(wantSigs))
		}

		for _, fn := range tr.Functions {
			if _, ok := wantSigs[fn.ResolvedSymbol]; !ok {
				t.Fatalf("unexpected symbol %s in %s", fn.ResolvedSymbol, path)
			}
			if _, ok := tr.Signatures[fn.ResolvedSymbol]; !ok {
				t.Fatalf("missing signature for %s in %s", fn.ResolvedSymbol, path)
			}
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

func containsTextSymbol(src, sym string) bool {
	return strings.Contains(src, "TEXT "+sym+"(SB)") ||
		strings.Contains(src, "TEXT "+sym+",") ||
		strings.Contains(src, "\nTEXT "+sym+"(SB)")
}

func checkSig(got, want extplan9asm.FuncSig) error {
	if got.Ret != want.Ret {
		return fmt.Errorf("ret mismatch: got %s, want %s", got.Ret, want.Ret)
	}
	if len(got.Args) != len(want.Args) {
		return fmt.Errorf("arg count mismatch: got %d, want %d", len(got.Args), len(want.Args))
	}
	for i := range got.Args {
		if got.Args[i] != want.Args[i] {
			return fmt.Errorf("arg[%d] mismatch: got %s, want %s", i, got.Args[i], want.Args[i])
		}
	}
	if len(got.ArgRegs) != len(want.ArgRegs) {
		return fmt.Errorf("arg reg count mismatch: got %d, want %d", len(got.ArgRegs), len(want.ArgRegs))
	}
	for i := range got.ArgRegs {
		if got.ArgRegs[i] != want.ArgRegs[i] {
			return fmt.Errorf("arg reg[%d] mismatch: got %s, want %s", i, got.ArgRegs[i], want.ArgRegs[i])
		}
	}
	return nil
}

func TestSigsForStdlibInternalBytealgArm64_CoversAllTextSymbols(t *testing.T) {
	if runtime.GOARCH != "arm64" {
		t.Skip("host is not arm64")
	}
	goroot := runtime.GOROOT()
	if goroot == "" {
		t.Skip("GOROOT not available")
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedTypesInfo | packages.NeedImports,
		Env:  os.Environ(),
	}
	pkgs, err := packages.LoadEx(nil, nil, cfg, "internal/bytealg")
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].Types == nil {
		t.Fatalf("load internal/bytealg: got %d pkgs, types=%v", len(pkgs), pkgs[0].Types)
	}
	pkg := pkgs[0]

	sfiles := []string{
		filepath.Join(goroot, "src", "internal", "bytealg", "compare_arm64.s"),
		filepath.Join(goroot, "src", "internal", "bytealg", "count_arm64.s"),
		filepath.Join(goroot, "src", "internal", "bytealg", "index_arm64.s"),
		filepath.Join(goroot, "src", "internal", "bytealg", "indexbyte_arm64.s"),
		filepath.Join(goroot, "src", "internal", "bytealg", "equal_arm64.s"),
	}
	sort.Strings(sfiles)

	for _, path := range sfiles {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		file, err := extplan9asm.Parse(extplan9asm.ArchARM64, string(src))
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		if len(file.Funcs) == 0 {
			t.Fatalf("unexpected empty funcs: %s", path)
		}

		tr, err := TranslateFileForPkg(pkg, path, runtime.GOOS, "arm64", nil)
		if err != nil {
			t.Fatalf("translate %s: %v", path, err)
		}
		if got, want := len(tr.Signatures), len(file.Funcs); got != want {
			t.Fatalf("%s: got %d signatures, want %d", path, got, want)
		}
	}
}
