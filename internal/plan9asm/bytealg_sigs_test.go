package plan9asm

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"
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
			requireBytealgOriginalFileContract(t, pkg, path, "linux", "arm64", wantSigs)
		})
	}
}

func requireBytealgOriginalFileContract(t *testing.T, pkg *packages.Package, path, goos, goarch string, want map[string]extplan9asm.FuncSig) {
	t.Helper()
	if filepath.Base(path) == "equal_"+goarch+".s" && (goarch == "386" || goarch == "amd64" || goarch == "arm64") {
		requireBytealgGCClosureBoundary(t, pkg, path, goos, goarch, want)
		return
	}
	tr := requireBytealgPublicTextRoots(t, pkg, path, goos, goarch)
	if want != nil {
		requireBytealgSignatures(t, tr, want)
	}
}

// The complete original gc equality file needs gc's hidden {entry, size}
// carrier. A Go signature alone does not supply it. LLGo's real Type*-bearing
// runtime bodies have separate owner/public-entry execution tests in build;
// they do not make this unmodified SDK assembly a supported closure entry.
func requireBytealgGCClosureBoundary(t *testing.T, pkg *packages.Package, path, goos, goarch string, want map[string]extplan9asm.FuncSig) {
	t.Helper()
	word, hiddenLoad, arch := extplan9asm.I64, "MOVQ 8(DX),BX", extplan9asm.ArchAMD64
	switch goarch {
	case "386":
		word, hiddenLoad = extplan9asm.I32, "MOVL 4(DX),BX"
	case "arm64":
		hiddenLoad, arch = "MOVD 8(R26),R2", extplan9asm.ArchARM64
	}
	expected := map[string]extplan9asm.FuncSig{
		"runtime.memequal":        {Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.Ptr, word}, Ret: extplan9asm.I1},
		"runtime.memequal_varlen": {Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.Ptr}, Ret: extplan9asm.I1},
	}
	if want != nil {
		if len(want) != len(expected) {
			t.Fatal("original equality signature inventory changed", want)
		}
		for name, sig := range expected {
			if err := checkSig(want[name], sig); err != nil {
				t.Fatalf("%s target signature: %v", name, err)
			}
		}
	}
	opt := TranslateOptions{GOROOT: runtime.GOROOT()}
	src, err := PreprocessFileForPkg(pkg, path, goos, goarch, opt)
	if err != nil {
		t.Fatal(err)
	}
	file, err := extplan9asm.Parse(arch, string(src))
	if err != nil {
		t.Fatal(err)
	}
	resolve := resolveSymFuncForTarget(pkg.PkgPath, goos, goarch)
	roots, hiddenLoads := map[string]bool{}, 0
	compact := func(text string) string { return strings.Join(strings.Fields(text), "") }
	for _, fn := range file.Funcs {
		if strings.HasSuffix(fn.Sym, "<>") {
			continue // Still retained in src and in the full translation below.
		}
		name := resolve(fn.Sym)
		if _, ok := expected[name]; !ok || roots[name] {
			t.Fatalf("duplicate or foreign original equality TEXT: %s", name)
		}
		if !shouldKeepResolvedFunc(pkg.PkgPath, goos, goarch, name) {
			t.Fatal("the adapter must retain every original public equality root", name)
		}
		roots[name] = true
		if name == "runtime.memequal_varlen" {
			for _, instruction := range fn.Instrs {
				if compact(instruction.Raw) == compact(hiddenLoad) {
					hiddenLoads++
				}
			}
		}
	}
	if len(roots) != len(expected) || hiddenLoads != 1 {
		t.Fatalf("incomplete original equality source: roots=%v, hidden loads=%d", roots, hiddenLoads)
	}
	owners := map[string]bool{}
	for _, file := range pkg.Syntax {
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || fn.Doc == nil {
				continue
			}
			for _, comment := range fn.Doc.List {
				fields := strings.Fields(comment.Text)
				if len(fields) != 3 || fields[0] != "//go:linkname" {
					continue
				}
				name := fields[2]
				wantSig, relevant := expected[name]
				if !relevant {
					continue
				}
				if fields[1] != fn.Name.Name || fn.Body != nil || owners[name] {
					t.Fatal("original equality declaration lost its unique bodyless linkname owner", name)
				}
				typed, ok := pkg.Types.Scope().Lookup(fn.Name.Name).(*types.Func)
				if !ok {
					t.Fatal("original equality declaration lacks its selected Go types", name)
				}
				sig, err := extplan9asm.DeriveGoFuncABI0Signature(name, typed, goarch, types.SizesFor("gc", goarch))
				if err != nil {
					t.Fatal(err)
				}
				if err := checkSig(sig, wantSig); err != nil {
					t.Fatalf("%s actual target declaration: %v", name, err)
				}
				owners[name] = true
			}
		}
	}
	if len(owners) != len(expected) {
		t.Fatal("original public equality signatures disappeared", owners)
	}
	opt.SourcePreprocessed = true
	tr, err := TranslateSourceModuleForPkgWithOptions(pkg, path, src, goos, goarch, opt)
	if tr != nil {
		t.Cleanup(tr.Module.Dispose)
		t.Fatal("unmodified gc equality unexpectedly acquired a closure entry")
	}
	if !errors.Is(err, extplan9asm.ErrProbeNeedsContext) || !strings.Contains(compact(err.Error()), compact(hiddenLoad)) {
		t.Fatalf("complete gc equality must reject the exact missing hidden load %q: %v", hiddenLoad, err)
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
			requireBytealgOriginalFileContract(t, pkg, path, "linux", "arm64", nil)
		})
	}
}
