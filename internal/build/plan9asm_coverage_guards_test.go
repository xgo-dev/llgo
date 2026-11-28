package build

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	llplan9asm "github.com/xgo-dev/llgo/internal/plan9asm"
	llssa "github.com/xgo-dev/llgo/ssa"
	llvm "github.com/xgo-dev/llvm"
	extplan9asm "github.com/xgo-dev/plan9asm"
)

func typecheckTestPkg(t *testing.T, pkgPath, src string) *packages.Package {
	t.Helper()
	dir := t.TempDir()
	goFile := filepath.Join(dir, "pkg.go")
	if err := os.WriteFile(goFile, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, goFile, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	sizes := types.SizesFor("gc", "arm64")
	typed, err := (&types.Config{Sizes: sizes, Importer: importer.Default()}).Check(pkgPath, fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	return &packages.Package{
		ID:              pkgPath,
		PkgPath:         pkgPath,
		Name:            typed.Name(),
		Dir:             dir,
		Fset:            fset,
		Types:           typed,
		TypesInfo:       info,
		TypesSizes:      sizes,
		Syntax:          []*ast.File{file},
		GoFiles:         []string{goFile},
		CompiledGoFiles: []string{goFile},
	}
}

func TestNativeAsmComponentsAndExportsGuards(t *testing.T) {
	ctx := &context{buildConf: &Config{Goos: "linux", Goarch: "arm64"}}

	// Unsupported target returns empty plan cleanly.
	if plan, err := planNativeAsmComponents(&context{buildConf: &Config{Goos: "js", Goarch: "wasm"}}, &packages.Package{}, nil); err != nil || len(plan.files) != 0 {
		t.Fatalf("wasm planNativeAsmComponents = %v, %v", plan, err)
	}

	// Conflicting go:cgo_import_dynamic in foreignNativeImports propagates error.
	conflictDynPkg := typecheckTestPkg(t, "example.com/dyn", `package dyn
//go:cgo_import_dynamic dup dup1 "libc.so"
//go:cgo_import_dynamic dup dup2 "libc.so"
var _ int
`)
	if _, err := planNativeAsmComponents(ctx, conflictDynPkg, []plan9AsmSource{
		{path: "a.s", source: []byte("TEXT callback<>(SB), NOSPLIT, $0\nJMP dup(SB)\n")},
	}); err == nil {
		t.Fatal("expected error for conflicting dynamic import")
	}
	// Valid foreignNativeImports path inside planNativeAsmComponents + captureSelectedPlan9Asm error path.
	validDynPkg := typecheckTestPkg(t, "example.com/dynok", `package dynok
//go:cgo_import_dynamic okfn strlen
var _ int
`)
	if plan, err := planNativeAsmComponents(ctx, validDynPkg, []plan9AsmSource{
		{path: "a.s", source: []byte("TEXT callback<>(SB), NOSPLIT, $0\nJMP okfn(SB)\n")},
	}); err != nil || plan.files["a.s"].Imports["okfn"] != "strlen" {
		t.Fatalf("valid foreignNativeImports plan failed: plan=%#v err=%v", plan, err)
	}
	if _, err := captureSelectedPlan9Asm(&context{buildConf: &Config{Goos: "linux", Goarch: "arm64"}}, validDynPkg, []string{filepath.Join(validDynPkg.Dir, "missing.s")}); err == nil {
		t.Fatal("expected captureSelectedPlan9Asm error for missing file")
	}

	// Parse error is skipped; duplicate TEXT, duplicate DATA owner across files,
	// initialized variable owned by DATA, duplicate GLOBL, non-uintptr DATA,
	// unreachable TEXT, and cross-file DATA/TEXT/pkgVar references.
	baseSrc := `package comp
import "unsafe"
var entryAddr uintptr
var badInit uintptr = 1
var nonPtr uint32
var pkgVar uint64
//go:linkname exportVar
var exportVar uint64
//go:linkname aliasedVar custom_sym
var aliasedVar uint64
//go:linkname aliasByte entry_code
var aliasByte byte
//go:linkname badCollision entry_code
var badCollision byte
func use() uintptr {
	_ = badInit
	_ = nonPtr
	_ = pkgVar
	_ = exportVar
	_ = aliasedVar
	_ = badCollision
	return uintptr(unsafe.Pointer(&aliasByte))
}
`
	pkg := typecheckTestPkg(t, "example.com/comp", baseSrc)

	// Invalid assembly parse error is skipped cleanly.
	if plan, err := planNativeAsmComponents(ctx, pkg, []plan9AsmSource{
		{path: "bad.s", source: []byte("TEXT bad(SB), NOSPLIT, $bad\n")},
	}); err != nil || len(plan.files) != 0 {
		t.Fatalf("invalid asm should skip cleanly: plan=%v err=%v", plan, err)
	}

	// Duplicate selected native TEXT error.
	if _, err := planNativeAsmComponents(ctx, pkg, []plan9AsmSource{
		{path: "a.s", source: []byte("TEXT helper(SB), NOSPLIT, $0-0\nRET\n")},
		{path: "b.s", source: []byte("TEXT helper(SB), NOSPLIT, $0-0\nRET\n")},
	}); err == nil {
		t.Fatal("expected duplicate selected native TEXT error")
	}

	// Duplicate native DATA owner across two files.
	if _, err := planNativeAsmComponents(ctx, pkg, []plan9AsmSource{
		{path: "a.s", source: []byte("GLOBL ·entryAddr(SB), $8\n")},
		{path: "b.s", source: []byte("GLOBL ·entryAddr(SB), $8\n")},
	}); err == nil {
		t.Fatal("expected duplicate native DATA owner error")
	}

	// Native DATA claiming an initialized Go variable.
	if _, err := planNativeAsmComponents(ctx, pkg, []plan9AsmSource{
		{path: "a.s", source: []byte("GLOBL ·badInit(SB), $8\n")},
	}); err == nil {
		t.Fatal("expected error for native DATA owning initialized Go variable")
	}

	// Native code object collides with another source alias (badCollision -> entry_code).
	if _, err := planNativeAsmComponents(ctx, pkg, []plan9AsmSource{
		{path: "a.s", source: []byte("TEXT entry_code(SB), NOSPLIT, $0-0\nRET\n")},
	}); err == nil {
		t.Fatal("expected collision error with second source alias")
	}

	// Cross-file data root + cross-file TEXT call + pkgVar/exportVar/aliasedVar imports + unreachable TEXT check.
	cleanPkg := typecheckTestPkg(t, "example.com/clean", `package clean
var entryAddr uintptr
var nonPtr uint32
var pkgVar uint64
//go:linkname exportVar
var exportVar uint64
//go:linkname aliasedVar custom_sym
var aliasedVar uint64
//go:linkname
//go:noinline
func keep() {
	_ = entryAddr
	_ = nonPtr
	_ = pkgVar
	_ = exportVar
	_ = aliasedVar
}
`)
	// Unreachable TEXT in selected file fails.
	if _, err := planNativeAsmComponents(ctx, cleanPkg, []plan9AsmSource{
		{path: "a.s", source: []byte("TEXT rootfn(SB), NOSPLIT, $0-0\nRET\nTEXT deadfn(SB), NOSPLIT, $0-0\nRET\nDATA ·entryAddr(SB)/8, $rootfn(SB)\nGLOBL ·entryAddr(SB), $8\n")},
	}); err == nil {
		t.Fatal("expected error for unreachable TEXT in selected file")
	}
	// Cross-file DATA root -> file B -> file C + cross-file GLOBL reference + pkgVar/exportVar/aliasedVar + non-uintptr DATA skipped.
	plan, err := planNativeAsmComponents(ctx, cleanPkg, []plan9AsmSource{
		{path: "data.s", source: []byte("DATA ·nonPtr(SB)/4, $1\nGLOBL ·nonPtr(SB), $4\nDATA ·entryAddr(SB)/8, $rootfn(SB)\nGLOBL ·entryAddr(SB), $8\n")},
		{path: "extra_data.s", source: []byte("DATA extTbl(SB)/8, $1\nGLOBL extTbl(SB), $8\n")},
		{path: "root.s", source: []byte("TEXT rootfn(SB), NOSPLIT, $0-0\nMOVD $·pkgVar(SB), R0\nMOVD $·exportVar(SB), R1\nMOVD $·aliasedVar(SB), R2\nMOVD $extTbl(SB), R3\nBL leaf(SB)\nBL ext_c(SB)\nRET\n")},
		{path: "leaf.s", source: []byte("TEXT leaf(SB), NOSPLIT, $0-0\nRET\n")},
	})
	if err != nil {
		t.Fatalf("cross-file component planning failed: %v", err)
	}
	if len(plan.files) != 4 || plan.files["data.s"].Imports["rootfn"] == "" || plan.files["root.s"].Imports["leaf"] == "" ||
		plan.files["root.s"].Imports["·aliasedVar"] != "custom_sym" || len(plan.unresolved["root.s"]) != 2 {
		t.Fatalf("unexpected cross-file plan: files=%#v unresolved=%#v", plan.files, plan.unresolved)
	}
	// nativeAsmOptions with valid nativeAddressEntries executes TranslateNativeSource (native_asm.go:73-74).
	if _, native, err := nativeAsmOptions(ctx, cleanPkg, []byte("TEXT rootfn(SB), NOSPLIT, $0-0\nRET\nDATA ·entryAddr(SB)/8, $rootfn(SB)\nGLOBL ·entryAddr(SB), NOPTR, $8\n")); err != nil || !native {
		t.Fatalf("nativeAsmOptions(address entry) = native=%v err=%v", native, err)
	}

	// nativeAsmAddressUses edge cases (non-uint8, non-type, non-uintptr, non-unsafe.Pointer, non-&, direct read).
	addrPkg := typecheckTestPkg(t, "example.com/addr", `package addr
import "unsafe"
var notByte uint16
var badRead byte
var badExpr byte
func helper(x any) uintptr { return 0 }
func ptrFn(x *byte) int { return 0 }
func check() uintptr {
	_ = int(1)
	_ = helper(&badExpr)
	_ = uintptr(1)
	_ = uintptr(int(1))
	_ = uintptr(ptrFn(&badExpr))
	_ = uintptr(unsafe.Pointer(nil))
	_ = badRead
	return uintptr(unsafe.Pointer(&badExpr))
}
`)
	if nativeAsmAddressUses(addrPkg, addrPkg.Types.Scope().Lookup("notByte").(*types.Var)) {
		t.Fatal("non-uint8 var should return false")
	}
	if nativeAsmAddressUses(addrPkg, addrPkg.Types.Scope().Lookup("badRead").(*types.Var)) {
		t.Fatal("directly read byte var should return false")
	}
	if nativeAsmAddressUses(addrPkg, addrPkg.Types.Scope().Lookup("badExpr").(*types.Var)) {
		t.Fatal("escaping &badExpr should return false")
	}

	// bindNativeAsmExports guards: empty plan, empty interested, conflicting providers, nil sig, missing provider, code entry conflict.
	emptyPlan := &nativeAsmPlan{}
	if err := bindNativeAsmExports(ctx, &aPackage{}, cleanPkg, emptyPlan); err != nil {
		t.Fatalf("empty plan failed: %v", err)
	}
	noInterestPlan := &nativeAsmPlan{files: map[string]extplan9asm.NativeOptions{"a.s": {}}}
	if err := bindNativeAsmExports(ctx, &aPackage{}, cleanPkg, noInterestPlan); err != nil {
		t.Fatalf("no-interest plan failed: %v", err)
	}
	// Missing provider error for unresolved target.
	if err := bindNativeAsmExports(&context{pkgs: map[*packages.Package]*aPackage{}}, &aPackage{}, cleanPkg, plan); err == nil {
		t.Fatal("expected missing provider error for ext_c")
	}
	// Nil export.sig error.
	depPkg := &packages.Package{PkgPath: "example.com/dep"}
	cleanPkg.Imports = map[string]*packages.Package{"example.com/dep": depPkg}
	depAPkg := &aPackage{
		Package: depPkg,
		linkSnapshot: &packageLinkSnapshot{
			cExports: []cExport{{goName: "ExtC", cName: "ext_c", sig: nil}},
		},
	}
	if err := bindNativeAsmExports(&context{pkgs: map[*packages.Package]*aPackage{depPkg: depAPkg}}, &aPackage{}, cleanPkg, plan); err == nil {
		t.Fatal("expected nil signature error for C export")
	}
	// Conflicting providers error (two deps exporting same C name with different Go owners).
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	depAPkg.linkSnapshot.cExports = []cExport{{goName: "ExtC1", cName: "ext_c", sig: sig}}
	dep2Pkg := &packages.Package{PkgPath: "example.com/dep2"}
	cleanPkg.Imports["example.com/dep2"] = dep2Pkg
	dep2APkg := &aPackage{
		Package: dep2Pkg,
		linkSnapshot: &packageLinkSnapshot{
			cExports: []cExport{{goName: "ExtC2", cName: "ext_c", sig: sig}},
		},
	}
	if err := bindNativeAsmExports(&context{pkgs: map[*packages.Package]*aPackage{depPkg: depAPkg, dep2Pkg: dep2APkg}}, &aPackage{}, cleanPkg, plan); err == nil {
		t.Fatal("expected conflicting native providers error")
	}
	// Native code object conflicting with selected C export.
	delete(cleanPkg.Imports, "example.com/dep2")
	conflictCodePlan := &nativeAsmPlan{
		files: map[string]extplan9asm.NativeOptions{
			"a.s": {CodeEntries: map[string]string{"fn": "ext_c"}, Imports: map[string]string{}},
		},
	}
	if err := bindNativeAsmExports(&context{pkgs: map[*packages.Package]*aPackage{depPkg: depAPkg}}, &aPackage{}, cleanPkg, conflictCodePlan); err == nil {
		t.Fatal("expected native object conflict with C export")
	}
	// Conflicting cgo_import_dynamic inside bindNativeAsmExports (lines 386-388).
	if err := bindNativeAsmExports(&context{pkgs: map[*packages.Package]*aPackage{}}, &aPackage{}, conflictDynPkg, &nativeAsmPlan{
		files:      map[string]extplan9asm.NativeOptions{"a.s": {Imports: map[string]string{}}},
		unresolved: map[string][]string{"a.s": {"dup"}},
	}); err == nil {
		t.Fatal("expected conflicting cgo_import_dynamic provider error")
	}
	// NeedRt scans runtime package exports in ctx.pkgs (lines 421-429).
	rtPkg := &packages.Package{PkgPath: "runtime"}
	rtAPkg := &aPackage{
		Package: rtPkg,
		linkSnapshot: &packageLinkSnapshot{
			cExports: []cExport{{goName: "RtExt", cName: "rt_sym", sig: sig}},
		},
	}
	rtPlan := &nativeAsmPlan{
		files:      map[string]extplan9asm.NativeOptions{"a.s": {Imports: map[string]string{}}},
		unresolved: map[string][]string{"a.s": {"rt_sym"}},
	}
	if err := bindNativeAsmExports(&context{pkgs: map[*packages.Package]*aPackage{rtPkg: rtAPkg}}, &aPackage{NeedRt: true}, cleanPkg, rtPlan); err != nil || rtPlan.files["a.s"].Imports["rt_sym"] != "rt_sym" {
		t.Fatalf("NeedRt runtime export binding failed: plan=%#v err=%v", rtPlan, err)
	}
}

func TestCgoFlagsAndPackageHelpersGuards(t *testing.T) {
	// splitCgoEnvFlags whitespace, quotes, and unterminated quote errors.
	if got, err := splitCgoEnvFlags(commandEnv{environ: []string{"K=  \t  "}}, "K"); err != nil || len(got) != 0 {
		t.Fatalf("whitespace-only split = %v, %v", got, err)
	}
	if got, err := splitCgoEnvFlags(commandEnv{environ: []string{`K=-DFOO 'a b' "c d"`}}, "K"); err != nil || len(got) != 3 {
		t.Fatalf("quoted split = %v, %v", got, err)
	}
	for _, key := range []string{"CGO_CPPFLAGS", "CGO_CFLAGS", "CGO_CXXFLAGS", "CGO_LDFLAGS"} {
		env := commandEnv{environ: []string{key + `="unterminated`}}
		if _, _, _, err := cgoCompilationFlags(env, nil, nil, true); err == nil {
			t.Fatalf("expected unterminated quote error for %s", key)
		}
	}

	// buildCgoSourcePackage nil and AltPkg branches.
	if got := buildCgoSourcePackage(nil, nil); got != nil {
		t.Fatalf("buildCgoSourcePackage(nil) = %v, want nil", got)
	}
	altFile := &ast.File{}
	mainPkg := &packages.Package{PkgPath: "main"}
	altPkg := &packages.Package{PkgPath: "alt", Syntax: []*ast.File{altFile}}
	ap := &aPackage{Package: mainPkg, AltPkg: &packages.Cached{Package: altPkg, Syntax: []*ast.File{altFile}}}
	if got := buildCgoSourcePackage(ap, []*ast.File{altFile}); got != altPkg {
		t.Fatalf("buildCgoSourcePackage(alt) = %v, want altPkg", got)
	}
	if got := buildCgoSourcePackage(ap, nil); got != mainPkg {
		t.Fatalf("buildCgoSourcePackage(main) = %v, want mainPkg", got)
	}

	// (*context).packageUsesOriginalCgo nil, AltPkg import "C", and missing GoFile error.
	ctx := &context{buildConf: &Config{}}
	if ok, err := ctx.packageUsesOriginalCgo(nil); err != nil || ok {
		t.Fatalf("packageUsesOriginalCgo(nil) = %v, %v", ok, err)
	}
	dir := t.TempDir()
	cgoFile := filepath.Join(dir, "c.go")
	if err := os.WriteFile(cgoFile, []byte("package alt\nimport \"C\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	altPkg.GoFiles = []string{cgoFile}
	if ok, err := ctx.packageUsesOriginalCgo(ap); err != nil || !ok {
		t.Fatalf("packageUsesOriginalCgo(alt with C) = %v, %v", ok, err)
	}
	badPkg := &aPackage{Package: &packages.Package{GoFiles: []string{filepath.Join(dir, "missing.go")}}}
	if _, err := ctx.packageUsesOriginalCgo(badPkg); err == nil {
		t.Fatal("expected error for missing GoFile in packageUsesOriginalCgo")
	}

	// parseCgoFilePreambles happy path (#cgo CPPFLAGS/CFLAGS) and error propagation.
	fset := token.NewFileSet()
	goodPreambleSrc := "package p\n// #cgo CPPFLAGS: -DFOO=1\n// #cgo CFLAGS: -O2\nimport \"C\"\n"
	goodParsed, err := parser.ParseFile(fset, "good.go", goodPreambleSrc, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	preambles, decls, err := parseCgoFilePreambles(commandEnv{}, nil, fset, []*ast.File{goodParsed}, "C")
	if err != nil || len(preambles) != 1 || len(decls) != 2 {
		t.Fatalf("parseCgoFilePreambles(good) = %d preambles, %d decls, %v", len(preambles), len(decls), err)
	}
	if cf, _, _ := collectCgoFlags(decls); len(cf) != 2 {
		t.Fatalf("collectCgoFlags = %v, want 2 flags", cf)
	}
	badPreambleSrc := "package p\n// #cgo unknownflag: -foo\nimport \"C\"\n"
	parsed, err := parser.ParseFile(fset, "bad.go", badPreambleSrc, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := parseCgoFilePreambles(commandEnv{}, nil, fset, []*ast.File{parsed}, "C"); err == nil {
		t.Fatal("expected parseCgoFilePreambles error for invalid #cgo directive")
	}
}

func TestAssemblyInputsNativeAsmAndFingerprintGuards(t *testing.T) {
	// llgoFileIsAssembly with -x flags and -xray- passthrough.
	ctx := &context{
		buildConf: &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH},
		crossCompile: crosscompile.Export{
			CC: "clang",
		},
	}
	if !llgoFileIsAssembly(ctx, []string{"-x", "assembler"}, "foo.c") {
		t.Fatal("expected -x assembler to mark .c as assembly")
	}
	if !llgoFileIsAssembly(ctx, []string{"-xassembler-with-cpp"}, "foo.c") {
		t.Fatal("expected -xassembler-with-cpp to mark .c as assembly")
	}
	if llgoFileIsAssembly(ctx, []string{"-xc"}, "foo.s") {
		t.Fatal("expected -xc to mark .s as non-assembly")
	}
	if !llgoFileIsAssembly(ctx, []string{"-xc", "-x", "none", "-xray-instrument"}, "foo.s") {
		t.Fatal("expected -x none to restore .s assembly classification")
	}

	// validateExplicitAssemblyOwners duplicate .s error.
	if err := validateExplicitAssemblyOwners([]llgoFileInput{{path: "a.c"}, {path: "x.s"}, {path: "./x.s"}}); err == nil {
		t.Fatal("expected duplicate explicit LLGoFiles assembly owner error")
	}

	// selectedOriginalCgoFiles deleted overlay, missing file, and syntax error.
	dir := t.TempDir()
	deleted := filepath.Join(dir, "del.go")
	badGo := filepath.Join(dir, "bad.go")
	_ = os.WriteFile(badGo, []byte("package p\nimport ("), 0o600)
	ctxOverlay := &context{buildConf: &Config{Overlay: map[string][]byte{deleted: nil}}}
	if _, _, err := selectedOriginalCgoFiles(ctxOverlay, &packages.Package{GoFiles: []string{deleted}}); err == nil {
		t.Fatal("expected deleted overlay error")
	}
	if _, _, err := selectedOriginalCgoFiles(ctxOverlay, &packages.Package{GoFiles: []string{badGo}}); err == nil {
		t.Fatal("expected parse error from selectedOriginalCgoFiles")
	}

	// cgoAssemblyInputs CGO_ENABLED=0, runtime/cgo non-gcc filtering, invalid preamble error, and duplicate explicit owner error.
	cgoGo := filepath.Join(dir, "cgo.go")
	_ = os.WriteFile(cgoGo, []byte("package p\nimport \"C\"\n"), 0o600)
	asmPath := filepath.Join(dir, "foo.S")
	_ = os.WriteFile(asmPath, []byte(".text\n"), 0o600)
	cgoPkg := &packages.Package{PkgPath: "example.com/cgo", Dir: dir, GoFiles: []string{cgoGo}, OtherFiles: []string{asmPath}}
	if inputs, err := cgoAssemblyInputs(&context{commands: commandEnv{environ: []string{"CGO_ENABLED=0"}}}, cgoPkg, nil); err != nil || inputs != nil {
		t.Fatalf("CGO_ENABLED=0 cgoAssemblyInputs = %v, %v", inputs, err)
	}
	rtCgoNonGcc := &packages.Package{PkgPath: "runtime/cgo", Dir: dir, GoFiles: []string{cgoGo}, OtherFiles: []string{filepath.Join(dir, "asm_amd64.s")}}
	if inputs, err := cgoAssemblyInputs(&context{buildConf: &Config{Goos: "linux", Goarch: "amd64"}}, rtCgoNonGcc, nil); err != nil || inputs != nil {
		t.Fatalf("runtime/cgo non-gcc assembly = %v, %v", inputs, err)
	}
	badCgoGo := filepath.Join(dir, "bad_cgo.go")
	_ = os.WriteFile(badCgoGo, []byte("package p\n// #cgo unknown: -x\nimport \"C\"\n"), 0o600)
	if _, err := cgoAssemblyInputs(&context{buildConf: &Config{Goos: "linux", Goarch: "amd64"}}, &packages.Package{PkgPath: "example.com/badcgo", Dir: dir, GoFiles: []string{badCgoGo}, OtherFiles: []string{asmPath}}, nil); err == nil {
		t.Fatal("expected parseCgoFilePreambles error in cgoAssemblyInputs")
	}
	if _, err := cgoAssemblyInputs(&context{buildConf: &Config{Goos: "linux", Goarch: "amd64"}}, cgoPkg, []llgoFileInput{{path: asmPath}}); err == nil {
		t.Fatal("expected duplicate cgo + explicit LLGoFiles owner error")
	}

	// externalizeNativeCodeAliases happy path & conflict error (non-i8 / thread-local / non-zero init).
	llvmCtx := llvm.NewContext()
	defer llvmCtx.Dispose()
	goMod := llvmCtx.NewModule("go")
	defer goMod.Dispose()
	natMod := llvmCtx.NewModule("nat")
	defer natMod.Dispose()
	fnTy := llvm.FunctionType(llvmCtx.VoidType(), nil, false)
	llvm.AddFunction(natMod, "ok_sym", fnTy)
	llvm.AddFunction(natMod, "bad_sym", fnTy)
	okGlobal := llvm.AddGlobal(goMod, llvmCtx.Int8Type(), "ok_sym")
	okGlobal.SetInitializer(llvm.ConstNull(llvmCtx.Int8Type()))
	if err := externalizeNativeCodeAliases(goMod, natMod, &nativeAsmPlan{aliases: map[string]string{"ok": "ok_sym", "missing": "no_fn"}}); err != nil {
		t.Fatalf("externalizeNativeCodeAliases(ok): %v", err)
	}
	if !okGlobal.IsDeclaration() {
		t.Fatal("expected ok_sym to become external declaration")
	}
	badGlobal := llvm.AddGlobal(goMod, llvmCtx.Int64Type(), "bad_sym")
	badGlobal.SetInitializer(llvm.ConstInt(llvmCtx.Int64Type(), 1, false))
	if err := externalizeNativeCodeAliases(goMod, natMod, &nativeAsmPlan{aliases: map[string]string{"bad": "bad_sym"}}); err == nil {
		t.Fatal("expected conflict error for non-i8/non-zero Go storage")
	}

	// digestPlan9AsmFiles & preprocessPlan9AsmForPkg cache hit and empty pkg.ID fallback.
	sfile := filepath.Join(dir, "ok_amd64.s")
	_ = os.WriteFile(sfile, []byte("TEXT ·F(SB),$0-0\nRET\n"), 0o600)
	fpCtx := &context{
		buildConf:       &Config{Goos: "linux", Goarch: "amd64"},
		plan9AsmSources: &sync.Map{},
		plan9asmReady:   true,
	}
	p9Pkg := &packages.Package{PkgPath: "example.com/fp", Dir: dir}
	opt := llplan9asm.TranslateOptions{GOROOT: runtime.GOROOT()}
	first, err := preprocessPlan9AsmForPkg(fpCtx, p9Pkg, sfile, opt)
	if err != nil {
		t.Fatal(err)
	}
	second, err := preprocessPlan9AsmForPkg(fpCtx, p9Pkg, sfile, opt)
	if err != nil || string(first) != string(second) {
		t.Fatalf("cached preprocess mismatch: err=%v", err)
	}
	if digests, err := digestPlan9AsmFiles(fpCtx, p9Pkg, nil); err != nil || digests != nil {
		t.Fatalf("empty digestPlan9AsmFiles = %v, %v", digests, err)
	}
	if _, err := digestPlan9AsmFiles(fpCtx, p9Pkg, []string{filepath.Join(dir, "missing.s")}); err == nil {
		t.Fatal("expected error for missing Plan 9 asm file")
	}
	// SourcePatchReplacesAsm skip (plan9asm_fingerprint.go:79-88) and !plan9asmEnabled && HasAltPkg skip (89-91).
	syscallCtx := &context{
		buildConf:       &Config{Goos: "linux", Goarch: "amd64"},
		conf:            &packages.Config{BuildFlags: []string{"-tags=llgo"}},
		patchFiles:      map[string][]string{"syscall": {"z_llgo_patch.go"}},
		plan9AsmSources: &sync.Map{},
		plan9asmReady:   true,
	}
	if d, err := digestPlan9AsmFiles(syscallCtx, &packages.Package{PkgPath: "syscall", Dir: dir}, []string{sfile}); err != nil || d != nil {
		t.Fatalf("SourcePatchReplacesAsm skip = %v, %v", d, err)
	}
	noP9Ctx := &context{
		buildConf:       &Config{Goos: "linux", Goarch: "amd64"},
		plan9AsmSources: &sync.Map{},
		plan9asmReady:   true,
		plan9asmMode:    plan9asmEnvNone,
	}
	if d, err := digestPlan9AsmFiles(noP9Ctx, &packages.Package{PkgPath: "runtime", Dir: dir}, []string{sfile}); err != nil || d != nil {
		t.Fatalf("disabled plan9asm HasAltPkg skip = %v, %v", d, err)
	}
	// digestPackagePlan9Asm with error and with AltPkg (plan9asm_fingerprint.go:25-43).
	fpCtx.sfilesCache = map[string][]string{"example.com/alt": {sfile}}
	fpCtx.llgoFilesCache = map[*packages.Package][]llgoFileInput{}
	altP9Pkg := &packages.Package{ID: "example.com/alt", PkgPath: "example.com/alt", Dir: dir, OtherFiles: []string{sfile}}
	if _, err := digestPackagePlan9Asm(fpCtx, &aPackage{Package: p9Pkg}, []string{filepath.Join(dir, "missing.s")}); err == nil {
		t.Fatal("expected digestPackagePlan9Asm error on missing primary asm")
	}
	if digests, err := digestPackagePlan9Asm(fpCtx, &aPackage{Package: p9Pkg, AltPkg: &packages.Cached{Package: altP9Pkg}}, nil); err != nil || len(digests) != 1 {
		t.Fatalf("digestPackagePlan9Asm(AltPkg) = %v, %v", digests, err)
	}
	badAltPkg := &packages.Package{ID: "example.com/badalt", PkgPath: "example.com/badalt", Dir: dir}
	fpCtx.sfilesCache["example.com/badalt"] = []string{filepath.Join(dir, "missing.s")}
	if _, err := digestPackagePlan9Asm(fpCtx, &aPackage{Package: p9Pkg, AltPkg: &packages.Cached{Package: badAltPkg}}, nil); err == nil {
		t.Fatal("expected digestPackagePlan9Asm error on missing alt asm")
	}
	// pkgSFiles propagating llgoPkgFileInputs error (build.go:4930-4931, plan9asm.go:519-520).
	errPkg := &packages.Package{ID: "example.com/errpkg", PkgPath: "example.com/errpkg", Dir: dir, GoFiles: []string{badCgoGo}, OtherFiles: []string{asmPath}}
	errCtx := &context{buildConf: &Config{Goos: "linux", Goarch: "amd64"}, llgoFilesCache: map[*packages.Package][]llgoFileInput{}}
	if _, err := pkgSFiles(errCtx, errPkg); err == nil {
		t.Fatal("expected pkgSFiles to propagate cgoAssemblyInputs error")
	}
}

func TestIsolatedPackageExclusiveOptLevelGuards(t *testing.T) {
	// nil buildConf branch.
	nilCtx := &context{}
	if got := nilCtx.isolatedPackageOptLevel(nil); got != optlevel.Unset {
		t.Fatalf("isolatedPackageOptLevel(nil buildConf) = %v, want Unset", got)
	}

	// Exercise newBackendSessionWithOptLevel and NewBackendProgramWithOptLevel(O0) directly,
	// plus tracePackageBuild exclusive class and executeIsolatedPackage O0 override branch.
	target := &llssa.Target{
		GOOS:     runtime.GOOS,
		GOARCH:   runtime.GOARCH,
		OptLevel: optlevel.Os,
	}
	prog := llssa.NewProgram(target)
	defer prog.Dispose()
	ctx := &context{
		prog: prog,
		mode: ModeBuild,
		buildConf: &Config{
			Goos:              runtime.GOOS,
			Goarch:            runtime.GOARCH,
			BuildMode:         BuildModeExe,
			OptLevel:          optlevel.Os,
			optLevelDefaulted: true,
		},
		patches:       cl.Patches{},
		sfilesCache:   map[string][]string{"example.com/heavy": nil},
		plan9asmReady: true,
	}
	session := ctx.newBackendSessionWithOptLevel(optlevel.O0)
	if session.prog == nil {
		t.Fatal("newBackendSessionWithOptLevel(O0) returned nil prog")
	}
	session.prog.Dispose()

	// Exercise executeIsolatedPackage O0 override branch cleanly with invalidEmbedPackage
	// and preparePackageGroup / tracePackageBuild / buildPackage exclusive FreeOSMemory path with unsafe.
	fset, badPkg := invalidEmbedPackage(t)
	badPkg.ssaInstructions = exclusiveIsolatedPackageSSACost
	ctx.conf = &packages.Config{Fset: fset}
	badTask := newPackageBuildTask(badPkg)
	badTask.isolated = true
	if got := ctx.isolatedPackageOptLevel(badTask); got != optlevel.O0 {
		t.Fatalf("isolatedPackageOptLevel(defaulted heavy) = %v, want O0", got)
	}
	if err := ctx.executeIsolatedPackage(badTask, false); err == nil {
		t.Fatal("expected frontend error from invalidEmbedPackage")
	}
	// Explicit optimization flags, LTO, and wasm keep the configured OptLevel.
	for _, tc := range []struct {
		name string
		conf Config
	}{
		{name: "explicit-O2", conf: Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH, OptLevel: optlevel.O2, optLevelDefaulted: false}},
		{name: "lto-enabled", conf: Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH, OptLevel: optlevel.Os, optLevelDefaulted: true, LTO: lto.Thin}},
		{name: "wasm-target", conf: Config{Goos: "wasip1", Goarch: "wasm", OptLevel: optlevel.Os, optLevelDefaulted: true}},
	} {
		probeCtx := &context{buildConf: &tc.conf}
		if got := probeCtx.isolatedPackageOptLevel(badTask); got != tc.conf.OptLevel {
			t.Fatalf("isolatedPackageOptLevel(%s) = %v, want %v", tc.name, got, tc.conf.OptLevel)
		}
	}

	unsafePkg := &aPackage{
		Package: &packages.Package{
			ID:      "unsafe",
			PkgPath: "unsafe",
			Types:   types.Unsafe,
		},
		ssaInstructions: exclusiveIsolatedPackageSSACost,
	}
	ctx.sfilesCache["unsafe"] = nil
	heavyTask := newPackageBuildTask(unsafePkg)
	if _, err := preparePackageGroup(ctx, []*packageBuildTask{heavyTask}, false); err != nil {
		t.Fatalf("preparePackageGroup(exclusive heavy unsafe) error = %v", err)
	}
	// Reset skip=false so estimatedBackendCost() >= exclusiveIsolatedPackageSSACost hits
	// buildPackage's exclusive FreeOSMemory path (package_build.go:250-252) and tracePackageBuild("exclusive").
	heavyTask.skip = false
	if err := tracePackageBuild(ctx, heavyTask, false, true, false); err != nil {
		t.Fatalf("tracePackageBuild(exclusive) error = %v", err)
	}
	// Also exercise preparePackageGroup's exclusive pre-FreeOSMemory path (package_build.go:185-187)
	// with a non-skipped heavy task (valid GoFiles, fails inside buildPkg after hitting line 186).
	badTask2 := newPackageBuildTask(badPkg)
	ctx.sfilesCache[badPkg.Package.ID] = nil
	if _, err := preparePackageGroup(ctx, []*packageBuildTask{badTask2}, false); err == nil {
		t.Fatal("expected preparePackageGroup to propagate invalidEmbedPackage error after exclusive FreeOSMemory")
	}
	// Also exercise tracePackageBuild class="patched" (package_build.go:205).
	ctx.patches["unsafe"] = cl.Patch{}
	if err := tracePackageBuild(ctx, heavyTask, false, true, false); err != nil {
		t.Fatalf("tracePackageBuild(patched) error = %v", err)
	}
}

func TestNativeAsmComponentsNeedRtErrorAndPlan9CompileGuards(t *testing.T) {
	// bindNativeAsmExports with NeedRt=true propagates runtime export errors.
	cleanPkg := typecheckTestPkg(t, "example.com/rtneed", "package rtneed\nvar _ int\n")
	rtPkg := &packages.Package{PkgPath: "runtime"}
	rtAPkg := &aPackage{
		Package: rtPkg,
		linkSnapshot: &packageLinkSnapshot{
			cExports: []cExport{{goName: "BadRt", cName: "bad_rt_sym", sig: nil}},
		},
	}
	rtPlan := &nativeAsmPlan{
		files:      map[string]extplan9asm.NativeOptions{"a.s": {Imports: map[string]string{}}},
		unresolved: map[string][]string{"a.s": {"bad_rt_sym"}},
	}
	if err := bindNativeAsmExports(&context{pkgs: map[*packages.Package]*aPackage{rtPkg: rtAPkg}}, &aPackage{NeedRt: true}, cleanPkg, rtPlan); err == nil || !strings.Contains(err.Error(), "no compiled signature") {
		t.Fatalf("bindNativeAsmExports(NeedRt nil sig) err = %v, want compiled signature error", err)
	}

	// compileSelectedPkgSFiles propagates planNativeAsmComponents errors.
	conflictPkg := typecheckTestPkg(t, "example.com/planerr", `package planerr
//go:cgo_import_dynamic dup dup1 "libc.so"
//go:cgo_import_dynamic dup dup2 "libc.so"
var _ int
`)
	conflictAsm := filepath.Join(conflictPkg.Dir, "a.s")
	if err := os.WriteFile(conflictAsm, []byte("TEXT callback<>(SB), 4, $0\nJMP dup(SB)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "arm64"})
	defer prog.Dispose()
	ctx := &context{
		prog:            prog,
		buildConf:       &Config{Goos: "linux", Goarch: "arm64"},
		commands:        commandEnv{dir: conflictPkg.Dir, environ: []string{"GOROOT=" + runtime.GOROOT()}},
		plan9AsmSources: &sync.Map{},
	}
	aPkg := &aPackage{Package: conflictPkg, LPkg: prog.NewPackage("planerr", conflictPkg.PkgPath)}
	if _, err := compileSelectedPkgSFiles(ctx, aPkg, conflictPkg, []string{conflictAsm}, false); err == nil || !strings.Contains(err.Error(), "conflicting dynamic import") {
		t.Fatalf("compileSelectedPkgSFiles(plan error) err = %v, want conflicting dynamic import", err)
	}

	// compileSelectedPkgSFiles disposes native module and returns error when
	// externalizeNativeCodeAliases finds conflicting non-zero Go storage.
	aliasPkg := typecheckTestPkg(t, "example.com/aliaserr", `package aliaserr
import "unsafe"
//go:linkname aliasByte entry_code
var aliasByte byte
func Addr() uintptr { return uintptr(unsafe.Pointer(&aliasByte)) }
`)
	aliasAsm := filepath.Join(aliasPkg.Dir, "entry.s")
	if err := os.WriteFile(aliasAsm, []byte("TEXT entry_code(SB), 4, $0-0\nRET\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	aliasAPkg := &aPackage{Package: aliasPkg, LPkg: prog.NewPackage("aliaserr", aliasPkg.PkgPath)}
	goMod := aliasAPkg.LPkg.Module()
	badGlobal := llvm.AddGlobal(goMod, goMod.Context().Int8Type(), "entry_code")
	badGlobal.SetInitializer(llvm.ConstInt(goMod.Context().Int8Type(), 1, false))
	ctx.commands.dir = aliasPkg.Dir
	if _, err := compileSelectedPkgSFiles(ctx, aliasAPkg, aliasPkg, []string{aliasAsm}, false); err == nil || !strings.Contains(err.Error(), "conflicts with Go storage") {
		t.Fatalf("compileSelectedPkgSFiles(alias conflict) err = %v, want Go storage conflict", err)
	}
}

func TestBuildCgoErrorAndCXXGuards(t *testing.T) {
	t.Run("packageUsesOriginalCgo error", func(t *testing.T) {
		dir := t.TempDir()
		file, pkg, ctx := buildCgoTestFile(t, dir, "")
		pkg.GoFiles = []string{filepath.Join(dir, "missing.go")}
		ctx.commands.environ = withEnv(ctx.commands.environ, "CGO_ENABLED=1")
		if _, _, err := buildCgo(ctx, pkg, []*ast.File{file}, nil, false); err == nil {
			t.Fatal("expected buildCgo to propagate packageUsesOriginalCgo error")
		}
	})

	t.Run("cgoCompilationFlags error", func(t *testing.T) {
		dir := t.TempDir()
		goFile := filepath.Join(dir, "cgo.go")
		src := "package demo\nimport \"C\"\n"
		if err := os.WriteFile(goFile, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, goFile, src, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		pkg := &aPackage{Package: &packages.Package{
			Fset:       fset,
			PkgPath:    "example.com/cgoerr",
			Dir:        dir,
			GoFiles:    []string{goFile},
			ExportFile: filepath.Join(dir, "demo.a"),
		}}
		ctx := &context{
			conf:      &packages.Config{},
			buildConf: &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH},
			commands: commandEnv{
				dir:     dir,
				environ: withEnv(os.Environ(), "CGO_ENABLED=1", `CGO_CPPFLAGS="-unterminated`),
			},
		}
		if _, _, err := buildCgo(ctx, pkg, []*ast.File{file}, nil, false); err == nil || !strings.Contains(err.Error(), "CGO_CPPFLAGS") {
			t.Fatalf("buildCgo(malformed CGO_CPPFLAGS) err = %v, want CGO_CPPFLAGS parse error", err)
		}
	})

	t.Run("sibling C++ source uses cxxflags", func(t *testing.T) {
		if _, err := exec.LookPath("clang++"); err != nil {
			if _, err2 := exec.LookPath("clang"); err2 != nil {
				t.Skip("clang/clang++ not found")
			}
		}
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "helper.cpp"), []byte("extern \"C\" int cpp_helper_val() { return CXX_MACRO; }\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		file, pkg, ctx := buildCgoTestFile(t, dir, "#cgo CXXFLAGS: -DCXX_MACRO=9\nint c_preamble_val = 1;")
		objects, _, err := buildCgo(ctx, pkg, []*ast.File{file}, nil, false)
		if err != nil {
			t.Fatalf("buildCgo with sibling .cpp failed: %v", err)
		}
		defer removeFiles(objects)
		if len(objects) != 2 {
			t.Fatalf("buildCgo(.cpp + preamble) objects = %v, want 2", objects)
		}
	})
}

func runFakeCCHelper(mode string) {
	var outPath string
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "-o" && i+1 < len(os.Args) {
			outPath = os.Args[i+1]
			break
		}
	}
	if outPath == "" {
		os.Exit(2)
	}
	switch mode {
	case "empty":
		if err := os.WriteFile(outPath, nil, 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(3)
		}
	case "dir":
		_ = os.Remove(outPath)
		if err := os.MkdirAll(outPath, 0o700); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(4)
		}
		if err := os.WriteFile(filepath.Join(outPath, "keep"), []byte("x"), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(5)
		}
	case "unreadable":
		if err := os.WriteFile(outPath, []byte("nonempty"), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(6)
		}
		if err := os.Chmod(outPath, 0o000); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(7)
		}
	default:
		os.Exit(8)
	}
}

func TestLLGoFilesReuseAndFingerprintErrorGuards(t *testing.T) {
	// llgoPkgFileInputs rejects duplicate explicit .s owners declared via LLGoFiles.
	dupPkg := typecheckTestPkg(t, "example.com/dupllgo", `package dupllgo
const LLGoFiles = "a.s; ./a.s"
`)
	if _, err := llgoPkgFileInputs(&context{buildConf: &Config{}}, dupPkg); err == nil || !strings.Contains(err.Error(), "duplicate explicit LLGoFiles owners") {
		t.Fatalf("llgoPkgFileInputs(duplicate .s) err = %v, want duplicate explicit owner error", err)
	}

	// reuseCompiledObject guards: missing file, directory, 0-byte file, and bad temp dir.
	dir := t.TempDir()
	if _, err := reuseCompiledObject("a.S", filepath.Join(dir, "missing.o")); err == nil {
		t.Fatal("expected reuseCompiledObject error for missing object")
	}
	if _, err := reuseCompiledObject("a.S", dir); err == nil {
		t.Fatal("expected reuseCompiledObject error for directory")
	}
	emptyObj := filepath.Join(dir, "empty.o")
	if err := os.WriteFile(emptyObj, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := reuseCompiledObject("a.S", emptyObj); err == nil {
		t.Fatal("expected reuseCompiledObject error for empty object")
	}
	goodObj := filepath.Join(dir, "good.o")
	if err := os.WriteFile(goodObj, []byte("objdata"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Run("reuseCompiledObject temp dir error", func(t *testing.T) {
		badTmp := filepath.Join(t.TempDir(), "no-such-tmp")
		t.Setenv("TMPDIR", badTmp)
		t.Setenv("TMP", badTmp)
		t.Setenv("TEMP", badTmp)
		if _, err := reuseCompiledObject("a.S", goodObj); err == nil {
			t.Fatal("expected reuseCompiledObject error when temp dir does not exist")
		}
	})

	// digestLLGoFileInputs error propagation for both assembly and C preprocessor paths.
	if _, err := exec.LookPath("clang"); err == nil {
		badAsm := filepath.Join(dir, "bad.S")
		badC := filepath.Join(dir, "bad.c")
		if err := os.WriteFile(badAsm, []byte(".llgo_invalid_asm_directive\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(badC, []byte("#error boom\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		fpCtx := &context{
			mode:              ModeBuild,
			buildConf:         &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH},
			commands:          commandEnv{dir: dir, environ: os.Environ()},
			llgoFileHashCache: make(map[llgoFileHashKey]string),
		}
		if _, err := digestLLGoFileInputs(fpCtx, []llgoFileInput{{path: badAsm}}, nil); err == nil {
			t.Fatal("expected digestLLGoFileInputs error for invalid assembly")
		}
		if _, err := digestLLGoFileInputs(fpCtx, []llgoFileInput{{path: badC}}, nil); err == nil || !strings.Contains(err.Error(), "preprocess file") {
			t.Fatalf("digestLLGoFileInputs(bad .c) err = %v, want preprocess error", err)
		}
	}

	// digestNativeAssemblyObject error branches: temp creation failure, empty object,
	// non-regular output directory (also exercises defer os.Remove error join), and unreadable object.
	asmSrc := filepath.Join(dir, "stub.S")
	if err := os.WriteFile(asmSrc, []byte(".text\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Run("digestNativeAssemblyObject temp dir error", func(t *testing.T) {
		badTmp := filepath.Join(t.TempDir(), "no-such-tmp")
		t.Setenv("TMPDIR", badTmp)
		t.Setenv("TMP", badTmp)
		t.Setenv("TEMP", badTmp)
		ctx := &context{buildConf: &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH}}
		if _, _, err := digestNativeAssemblyObject(ctx, asmSrc, nil, false); err == nil || !strings.Contains(err.Error(), "create assembly fingerprint object") {
			t.Fatalf("digestNativeAssemblyObject(bad temp) err = %v, want create error", err)
		}
	})

	fakeCC := writeBuildTestTool(t, t.TempDir(), "clang")
	for _, mode := range []string{"empty", "dir", "unreadable"} {
		t.Run("fakeCC-"+mode, func(t *testing.T) {
			if mode == "unreadable" && (runtime.GOOS == "windows" || os.Geteuid() == 0) {
				t.Skip("chmod 0000 unreadable file test requires non-root Unix")
			}
			tmpDir := t.TempDir()
			t.Setenv("TMPDIR", tmpDir)
			t.Setenv("TMP", tmpDir)
			t.Setenv("TEMP", tmpDir)
			t.Cleanup(func() {
				entries, _ := os.ReadDir(tmpDir)
				for _, e := range entries {
					p := filepath.Join(tmpDir, e.Name())
					_ = os.Chmod(p, 0o700)
					_ = os.RemoveAll(p)
				}
			})
			ctx := &context{
				buildConf:    &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH},
				crossCompile: crosscompile.Export{CC: fakeCC},
				commands:     commandEnv{dir: dir, environ: withEnv(os.Environ(), "LLGO_TEST_FAKE_CC_HELPER="+mode)},
			}
			_, _, err := digestNativeAssemblyObject(ctx, asmSrc, nil, false)
			if err == nil {
				t.Fatalf("digestNativeAssemblyObject(%s) unexpectedly succeeded", mode)
			}
			if mode == "dir" && !strings.Contains(err.Error(), "remove assembly fingerprint object") {
				t.Fatalf("digestNativeAssemblyObject(dir) err = %v, want joined remove error", err)
			}
		})
	}
}

func TestCollectAndUnusedGlobalInitErrorGuards(t *testing.T) {
	// collectPackageInputs propagates packageUsesOriginalCgo error when CGO_ENABLED=1
	// and a selected Go source has invalid syntax (file hashing succeeds, import parse fails).
	dir := t.TempDir()
	badGo := filepath.Join(dir, "bad.go")
	if err := os.WriteFile(badGo, []byte("package p\nimport ("), 0o600); err != nil {
		t.Fatal(err)
	}
	pkg := &packages.Package{
		ID:      "example.com/badcgo",
		PkgPath: "example.com/badcgo",
		Dir:     dir,
		GoFiles: []string{badGo},
	}
	ctx := &context{
		mode:           ModeGen,
		buildConf:      &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH},
		commands:       commandEnv{dir: dir, environ: []string{"CGO_ENABLED=1"}},
		sfilesCache:    map[string][]string{pkg.ID: nil},
		llgoFilesCache: map[*packages.Package][]llgoFileInput{pkg: nil},
	}
	m := newManifestBuilder()
	if err := ctx.collectPackageInputs(m, &aPackage{Package: pkg}); err == nil || !strings.Contains(err.Error(), "inspect cgo inputs") {
		t.Fatalf("collectPackageInputs(malformed Go) err = %v, want inspect cgo inputs error", err)
	}

	// collectAsmGlobalInitReferences returns false when captureSelectedPlan9Asm fails.
	badAsm := filepath.Join(dir, "bad.s")
	if err := os.WriteFile(badAsm, []byte("#include \"missing-init-header.h\"\nTEXT ·F(SB),$0-0\nRET\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	asmPkg := &packages.Package{
		ID:         "example.com/badinitasm",
		PkgPath:    "example.com/badinitasm",
		Dir:        dir,
		OtherFiles: []string{badAsm},
	}
	asmCtx := &context{
		buildConf:       &Config{Goos: "linux", Goarch: "amd64"},
		commands:        commandEnv{dir: dir, environ: []string{"GOROOT=" + runtime.GOROOT()}},
		sfilesCache:     map[string][]string{asmPkg.ID: {badAsm}},
		plan9asmReady:   true,
		plan9asmMode:    plan9asmEnvAll,
		plan9AsmSources: &sync.Map{},
	}
	if ok := collectAsmGlobalInitReferences(asmCtx, asmPkg, map[string]bool{}, nil); ok {
		t.Fatal("collectAsmGlobalInitReferences unexpectedly returned true on missing include")
	}
}
