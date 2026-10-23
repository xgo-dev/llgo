package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
)

// Match cmd/go's compiler ownership, not the case of the extension. In
// runtime/cgo only gcc_* belongs to the native compiler; asm_* is still Plan 9.
func TestPackageAssemblyRuntimeCgoMixedRouting(t *testing.T) {
	for _, operation := range []string{"Plan9 owners", "native owners", "fingerprint"} {
		t.Run(operation, func(t *testing.T) {
			ctx, pkg := assemblyLanguageTestFixture(t, "runtime/cgo", true, false)
			plan9 := assemblyLanguageTestWrite(t, pkg.Dir, "asm_amd64.s", "#include \"body.h\"\n")
			header := assemblyLanguageTestWrite(t, pkg.Dir, "body.h", "#define VALUE 1\nTEXT ·Foo(SB),4,$0-0\nMOVQ $VALUE, AX\nRET\n")
			native := assemblyLanguageTestWrite(t, pkg.Dir, "gcc_amd64.S", "#if defined(__APPLE__)\n#define ENTRY _probe\n#else\n#define ENTRY probe\n#endif\n.text\n.globl ENTRY\nENTRY:\nret\n")
			pkg.OtherFiles = []string{plan9, header, native}
			switch operation {
			case "Plan9 owners":
				files, err := pkgSFiles(ctx, pkg)
				if err != nil {
					t.Fatal(err)
				}
				assemblyLanguageTestPaths(t, files, []string{plan9})
			case "native owners":
				assemblyLanguageTestPaths(t, assemblyLanguageTestNativeOwners(t, ctx, pkg), []string{native})
			case "fingerprint":
				manifest := newManifestBuilder()
				if err := ctx.collectPackageInputs(manifest, &aPackage{Package: pkg}); err != nil {
					t.Fatalf("native #if entered Plan 9 preprocessing: %v", err)
				}
				if len(manifest.pkg.Plan9AsmFiles) != 1 || manifest.pkg.Plan9AsmFiles[0].Path != plan9 {
					t.Fatalf("wrong Plan 9 fingerprint owners: %+v", manifest.pkg.Plan9AsmFiles)
				}
				var other []string
				for _, input := range manifest.pkg.OtherFiles {
					other = append(other, input.Path)
				}
				if !slices.Contains(other, native) {
					t.Fatal("routing native assembly silently removed its cache input")
				}
			}
		})
	}
}

func TestPackageAssemblyCgoNativeOwnersRetainBothExtensions(t *testing.T) {
	ctx, pkg := assemblyLanguageTestFixture(t, "example.com/cgoasm", true, false)
	lower := assemblyLanguageTestWrite(t, pkg.Dir, "native_amd64.s", ".text\n.globl lower\nlower:\nret\n")
	upper := assemblyLanguageTestWrite(t, pkg.Dir, "native_amd64.S", "#if defined(__ELF__)\n.text\n#endif\n.globl upper\nupper:\nret\n")
	// A directory glob must not invent a second target's compiler input.
	assemblyLanguageTestWrite(t, pkg.Dir, "native_arm64.S", "not selected for amd64\n")
	pkg.OtherFiles = []string{lower, upper}
	t.Run("no Plan9 owner", func(t *testing.T) {
		files, err := pkgSFiles(ctx, pkg)
		if err != nil {
			t.Fatal(err)
		}
		assemblyLanguageTestPaths(t, files, nil)
	})
	t.Run("actual native producer owns both", func(t *testing.T) {
		// LLGoFiles inputs supply concatPkgLinkFiles' real clFile loop, rather
		// than merely deleting files from Plan 9 and counting a skip.
		assemblyLanguageTestPaths(t, assemblyLanguageTestNativeOwners(t, ctx, pkg), []string{lower, upper})
	})
}

func TestPackageAssemblyLanguageUsesSelectedOriginalGoSources(t *testing.T) {
	for _, test := range []struct {
		name                  string
		diskCgo, generatedCgo bool
		overlay               *bool
		wantCgo, deleted      bool
	}{
		{name: "original C import survives generated syntax", diskCgo: true, wantCgo: true},
		{name: "generated C import is not original selection", generatedCgo: true},
		{name: "overlay adds C import", overlay: assemblyLanguageTestBool(true), wantCgo: true},
		{name: "overlay removes C import", diskCgo: true, generatedCgo: true, overlay: assemblyLanguageTestBool(false)},
		{name: "deleted selected Go overlay is not disk fallback", diskCgo: true, deleted: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, pkg := assemblyLanguageTestFixture(t, "example.com/selected", test.diskCgo, test.generatedCgo)
			asm := assemblyLanguageTestWrite(t, pkg.Dir, "entry_amd64.s", "#include \"body.h\"\n")
			pkg.OtherFiles = []string{asm}
			// Not present in GoFiles, so this inactive C import cannot change
			// ownership even though it is on disk next to selected sources.
			assemblyLanguageTestWrite(t, pkg.Dir, "ignored.go", "//go:build ignore\n\npackage fixture\nimport \"C\"\n")
			if test.overlay != nil {
				ctx.buildConf.Overlay[pkg.GoFiles[0]] = []byte(assemblyLanguageTestGoSource(*test.overlay))
			}
			if test.deleted {
				ctx.buildConf.Overlay[pkg.GoFiles[0]] = nil
			}
			files, err := pkgSFiles(ctx, pkg)
			if test.deleted {
				if err == nil {
					t.Fatal("deleted original Go source silently fell back to disk during classification")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var want []string
			if !test.wantCgo {
				want = []string{asm}
			}
			assemblyLanguageTestPaths(t, files, want)
		})
	}
}

func TestPackageAssemblyPlainPlan9IncludeOnlyRemainsObserved(t *testing.T) {
	ctx, pkg := assemblyLanguageTestFixture(t, "example.com/plan9", false, false)
	asm := assemblyLanguageTestWrite(t, pkg.Dir, "entry_amd64.s", "#include \"body.h\"\n")
	header := assemblyLanguageTestWrite(t, pkg.Dir, "body.h", "#define VALUE 1\nTEXT ·Foo(SB),4,$0-0\nMOVQ $VALUE, AX\nRET\n")
	pkg.OtherFiles = []string{asm, header}
	files, err := pkgSFiles(ctx, pkg)
	if err != nil {
		t.Fatal(err)
	}
	assemblyLanguageTestPaths(t, files, []string{asm})
	assemblyLanguageTestPaths(t, assemblyLanguageTestNativeOwners(t, ctx, pkg), nil)
	fingerprint := func() string {
		t.Helper()
		manifest := newManifestBuilder()
		if err := ctx.collectPackageInputs(manifest, &aPackage{Package: pkg}); err != nil {
			t.Fatal(err)
		}
		if len(manifest.pkg.Plan9AsmFiles) != 1 || manifest.pkg.Plan9AsmFiles[0].Path != asm {
			t.Fatalf("include-only Plan 9 source was dropped: %+v", manifest.pkg.Plan9AsmFiles)
		}
		return manifest.Fingerprint()
	}
	before := fingerprint()
	info, err := os.Stat(header)
	if err != nil {
		t.Fatal(err)
	}
	assemblyLanguageTestWrite(t, pkg.Dir, "body.h", "#define VALUE 2\nTEXT ·Foo(SB),4,$0-0\nMOVQ $VALUE, AX\nRET\n")
	if err := os.Chtimes(header, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if before == fingerprint() {
		t.Fatal("included Plan 9 body changed without changing its expanded fingerprint")
	}
	sigs, err := plan9asmSigsForPkg(ctx, pkg.PkgPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, present := sigs[pkg.PkgPath+".Foo"]; !present {
		t.Fatalf("include-only public TEXT disappeared from signatures: %v", sigs)
	}
}

func assemblyLanguageTestFixture(t *testing.T, path string, originalCgo, generatedCgo bool) (*context, *packages.Package) {
	t.Helper()
	dir := t.TempDir()
	original := assemblyLanguageTestWrite(t, dir, "original.go", assemblyLanguageTestGoSource(originalCgo))
	generated := assemblyLanguageTestWrite(t, dir, "generated.go", assemblyLanguageTestGoSource(generatedCgo))
	fset := token.NewFileSet()
	syntax, err := parser.ParseFile(fset, generated, nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	typesPkg := types.NewPackage(path, "fixture")
	typesPkg.Scope().Insert(types.NewFunc(token.NoPos, typesPkg, "Foo", types.NewSignatureType(nil, nil, nil, nil, nil, false)))
	pkg := &packages.Package{ID: path, PkgPath: path, Name: "fixture", Dir: dir,
		GoFiles: []string{original}, CompiledGoFiles: []string{generated},
		Fset: fset, Syntax: []*ast.File{syntax}, Types: typesPkg}
	goroot := t.TempDir()
	assemblyLanguageTestWrite(t, goroot, "VERSION", "go1.27.2\n")
	return &context{
		mode: ModeBuild, plan9asmReady: true, plan9asmMode: plan9asmEnvAll,
		conf: &packages.Config{}, buildConf: &Config{Goos: "linux", Goarch: "amd64", Overlay: map[string][]byte{}},
		commands: commandEnv{dir: dir, environ: []string{"GOROOT=" + goroot}},
		pkgs:     map[*packages.Package]Package{pkg: {Package: pkg}},
	}, pkg
}

func assemblyLanguageTestGoSource(cgo bool) string {
	if cgo {
		return "package fixture\nimport \"C\"\n"
	}
	return "package fixture\nimport \"unsafe\"\nvar _ unsafe.Pointer\n"
}

func assemblyLanguageTestNativeOwners(t *testing.T, ctx *context, pkg *packages.Package) []string {
	t.Helper()
	inputs, err := llgoPkgFileInputs(ctx, pkg)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, input := range inputs {
		paths = append(paths, input.path)
	}
	return paths
}

func assemblyLanguageTestPaths(t *testing.T, got, want []string) {
	t.Helper()
	got, want = slices.Clone(got), slices.Clone(want)
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("assembly compiler owners = %q, want %q", got, want)
	}
}

func assemblyLanguageTestWrite(t *testing.T, dir, name, contents string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func assemblyLanguageTestBool(value bool) *bool { return &value }
