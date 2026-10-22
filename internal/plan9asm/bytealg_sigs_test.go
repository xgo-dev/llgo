package plan9asm

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
	"github.com/xgo-dev/llvm"
	extplan9asm "github.com/xgo-dev/plan9asm"
)

func TestSigsForStdlibInternalBytealgArm64(t *testing.T) {
	goroot := runtime.GOROOT()
	if goroot == "" {
		t.Fatal("GOROOT not available")
	}
	pkg := loadStdlibInternalBytealgForTarget(t, "linux", "arm64")

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
	for path, wantSigs := range sfiles {
		t.Run(filepath.Base(path), func(t *testing.T) {
			tr := requireBytealgPublicTextRoots(t, pkg, path, "linux", "arm64")
			requireBytealgSignatures(t, tr, wantSigs)
		})
	}
}

func requireBytealgSignatures(t *testing.T, tr *ModuleTranslation, want map[string]extplan9asm.FuncSig) {
	t.Helper()
	public := 0
	for _, fn := range tr.Functions {
		if strings.HasSuffix(fn.TextSymbol, "<>") {
			continue // An explicitly declared custom helper is not a Go root.
		}
		public++
		expected, ok := want[fn.ResolvedSymbol]
		if !ok {
			t.Fatalf("unexpected public root %s", fn.ResolvedSymbol)
		}
		if err := checkSig(tr.Signatures[fn.ResolvedSymbol], expected); err != nil {
			t.Fatalf("%s: %v", fn.ResolvedSymbol, err)
		}
	}
	if public != len(want) {
		t.Fatalf("public root count = %d, want %d", public, len(want))
	}
}

// A private <> TEXT remains part of the complete source continuation proof,
// not an independently callable Go signature. Compare the full original public
// root inventory, rather than equating all physical TEXT nodes with Go entries.
func requireBytealgPublicTextRoots(t *testing.T, pkg *packages.Package, path, goos, goarch string) *ModuleTranslation {
	t.Helper()
	opt := TranslateOptions{GOROOT: runtime.GOROOT()}
	src, err := PreprocessFileForPkg(pkg, path, goos, goarch, opt)
	if err != nil {
		t.Fatal(err)
	}
	arch := extplan9asm.ArchAMD64 // The parser's shared x86 grammar includes 386.
	if goarch == "arm64" {
		arch = extplan9asm.ArchARM64
	}
	file, err := extplan9asm.Parse(arch, string(src))
	if err != nil {
		t.Fatal(err)
	}
	resolve := resolveSymFuncForTarget(pkg.PkgPath, goos, goarch)
	roots := map[string]bool{}
	private := map[string]string{}
	for _, fn := range file.Funcs {
		name := resolve(fn.Sym)
		if strings.HasSuffix(fn.Sym, "<>") {
			private[fn.Sym] = name
			continue
		}
		if roots[name] {
			t.Fatalf("duplicate original public TEXT root %s", name)
		}
		roots[name] = true
	}
	if len(roots) == 0 {
		t.Fatal("empty original public TEXT root inventory")
	}
	opt.SourcePreprocessed = true
	tr, err := TranslateSourceModuleForPkgWithOptions(pkg, path, src, goos, goarch, opt)
	if err != nil {
		t.Fatalf("translate %s: %v", path, err)
	}
	t.Cleanup(tr.Module.Dispose)
	if err := llvm.VerifyModule(tr.Module, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid translated module: %v", err)
	}
	seen := map[string]bool{}
	for _, fn := range tr.Functions {
		if strings.HasSuffix(fn.TextSymbol, "<>") {
			if name, ok := private[fn.TextSymbol]; !ok || name != fn.ResolvedSymbol {
				t.Fatalf("private helper metadata lacks its original source identity: %+v", fn)
			}
			continue
		}
		if !roots[fn.ResolvedSymbol] || seen[fn.ResolvedSymbol] {
			t.Fatalf("duplicate or foreign public Go entry: %+v", fn)
		}
		seen[fn.ResolvedSymbol] = true
		if _, ok := tr.Signatures[fn.ResolvedSymbol]; !ok {
			t.Fatalf("missing public root signature: %s", fn.ResolvedSymbol)
		}
		if entry := tr.Module.NamedFunction(fn.ResolvedSymbol); entry.IsNil() || entry.BasicBlocksCount() == 0 {
			t.Fatalf("missing public root definition: %s", fn.ResolvedSymbol)
		}
	}
	if len(seen) != len(roots) {
		t.Fatalf("public root union incomplete: got %v, want %v", seen, roots)
	}
	return tr
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
	goroot := runtime.GOROOT()
	if goroot == "" {
		t.Fatal("GOROOT not available")
	}
	pkg := loadStdlibInternalBytealgForTarget(t, "linux", "arm64")

	sfiles := []string{
		filepath.Join(goroot, "src", "internal", "bytealg", "compare_arm64.s"),
		filepath.Join(goroot, "src", "internal", "bytealg", "count_arm64.s"),
		filepath.Join(goroot, "src", "internal", "bytealg", "index_arm64.s"),
		filepath.Join(goroot, "src", "internal", "bytealg", "indexbyte_arm64.s"),
		filepath.Join(goroot, "src", "internal", "bytealg", "equal_arm64.s"),
	}
	sort.Strings(sfiles)

	for _, path := range sfiles {
		t.Run(filepath.Base(path), func(t *testing.T) {
			requireBytealgPublicTextRoots(t, pkg, path, "linux", "arm64")
		})
	}
}
