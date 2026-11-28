package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	extplan9asm "github.com/xgo-dev/plan9asm"
)

// Compile the real export declaration producer. A plain LLVM function name or
// a package that merely happens to be present in ctx.pkgs is not authority.
func TestPlan9AsmNativeComponentCExportProviders(t *testing.T) {
	files := make(map[string]string)
	for _, name := range []string{"go.mod", "provider.go", "provider.s"} {
		contents, err := os.ReadFile(filepath.Join("testdata", "plan9asm-native-provider", name))
		if err != nil {
			t.Fatal(err)
		}
		files[name] = string(contents)
	}
	directory := nativeAddressFixtureWrite(t, files)
	conf := NewDefaultConf(ModeGen)
	compiled, err := Build(Invocation{Dir: directory, Args: []string{"."}, Config: conf})
	if err != nil || len(compiled) != 1 {
		t.Fatalf("actual C export producer: %v, %v", compiled, err)
	}
	provider := compiled[0]
	t.Cleanup(provider.LPkg.Prog.Dispose)
	exports := packageCExports(provider)
	if len(exports) != 1 || exports[0].cName != "native_component_export" || exports[0].sig == nil {
		t.Fatalf("actual compiler C export snapshot: %+v", exports)
	}
	ordinary := strings.TrimSuffix(exports[0].goName, "native_component_export") + "Ordinary"
	if provider.LPkg.FuncOf(ordinary) == nil {
		t.Fatalf("the non-C Go function %s was not actually compiled", ordinary)
	}
	ctx := &context{pkgs: map[*packages.Package]Package{provider.Package: provider}}
	plan := func(target string) *nativeAsmPlan {
		return &nativeAsmPlan{
			files: map[string]extplan9asm.NativeOptions{
				"provider.s": {Imports: make(map[string]string)},
			},
			unresolved: map[string][]string{"provider.s": {target}},
		}
	}
	selected := plan("native_component_export")
	if err := bindNativeAsmExports(ctx, provider, provider.Package, selected); err != nil {
		t.Fatal(err)
	}
	if selected.files["provider.s"].Imports["native_component_export"] != "native_component_export" {
		t.Fatal("the actual selected C export was not bound")
	}
	for _, target := range []string{"Ordinary", "missing_native_export"} {
		if err := bindNativeAsmExports(ctx, provider, provider.Package, plan(target)); err == nil {
			t.Fatalf("unexported/unknown symbol %s acquired a native C role", target)
		}
	}
	// This context still contains the genuine compiled provider. The consumer
	// does not import it, so the compiler must not search unrelated packages.
	consumerType := types.NewPackage("example.com/unrelated-consumer", "consumer")
	consumer := &aPackage{Package: &packages.Package{PkgPath: consumerType.Path(), Types: consumerType}}
	ctx.pkgs[consumer.Package] = consumer
	if err := bindNativeAsmExports(ctx, consumer, consumer.Package, plan("native_component_export")); err == nil {
		t.Fatal("an unselected package supplied a native symbol")
	}
}

// These are actual separately compiled export producers, not symbol-name
// guesses. Reaching the same provider twice is allowed; competing Go owners
// and inconsistent explicit dynamic imports cannot silently overwrite it.
func TestPlan9AsmNativeComponentProviderConflicts(t *testing.T) {
	compile := func(module string) *aPackage {
		t.Helper()
		files := make(map[string]string)
		for _, name := range []string{"go.mod", "provider.go", "provider.s"} {
			contents, err := os.ReadFile(filepath.Join("testdata", "plan9asm-native-provider", name))
			if err != nil {
				t.Fatal(err)
			}
			files[name] = string(contents)
		}
		files["go.mod"] = "module " + module + "\n\ngo 1.20\n"
		directory := nativeAddressFixtureWrite(t, files)
		compiled, err := Build(Invocation{Dir: directory, Args: []string{"."}, Config: NewDefaultConf(ModeGen)})
		if err != nil || len(compiled) != 1 {
			t.Fatalf("compile actual provider %s: %v, %v", module, compiled, err)
		}
		t.Cleanup(compiled[0].LPkg.Prog.Dispose)
		return compiled[0]
	}
	first := compile("example.com/component-provider-first")
	second := compile("example.com/component-provider-second")
	for _, pkg := range []*aPackage{first, second} {
		exports := packageCExports(pkg)
		if len(exports) != 1 || exports[0].cName != "native_component_export" || exports[0].sig == nil {
			t.Fatalf("actual compiled export identity: %+v", exports)
		}
	}
	firstExport, secondExport := packageCExports(first)[0], packageCExports(second)[0]
	if firstExport.goName == secondExport.goName {
		t.Fatal("independent module producers do not have distinct Go owners")
	}
	plan := func(target string) *nativeAsmPlan {
		return &nativeAsmPlan{
			files:      map[string]extplan9asm.NativeOptions{"consumer.s": {Imports: make(map[string]string)}},
			unresolved: map[string][]string{"consumer.s": {target}},
		}
	}
	consumer := func(source string, dependencies ...*aPackage) (*context, *aPackage) {
		t.Helper()
		parsed, err := parser.ParseFile(token.NewFileSet(), "consumer.go", source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		typ := types.NewPackage("example.com/component-consumer", "consumer")
		pkg := &aPackage{Package: &packages.Package{
			PkgPath: typ.Path(), Types: typ, Syntax: []*ast.File{parsed},
			Imports: make(map[string]*packages.Package),
		}}
		ctx := &context{pkgs: map[*packages.Package]Package{pkg.Package: pkg}}
		for _, dependency := range dependencies {
			pkg.Imports[dependency.PkgPath] = dependency.Package
			ctx.pkgs[dependency.Package] = dependency
		}
		return ctx, pkg
	}
	t.Run("repeated same owner", func(t *testing.T) {
		ctx, pkg := consumer("package consumer", first)
		pkg.Imports["second-path-to-same-provider"] = first.Package
		if err := bindNativeAsmExports(ctx, pkg, pkg.Package, plan(firstExport.cName)); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("different actual owners", func(t *testing.T) {
		ctx, pkg := consumer("package consumer", first, second)
		if err := bindNativeAsmExports(ctx, pkg, pkg.Package, plan(firstExport.cName)); err == nil {
			t.Fatal("different compiled Go owners silently provided the same native symbol")
		}
	})
	t.Run("conflicting explicit dynamic import", func(t *testing.T) {
		ctx, pkg := consumer("package consumer\n//go:cgo_import_dynamic native_component_export different_external\n", first)
		if err := bindNativeAsmExports(ctx, pkg, pkg.Package, plan(firstExport.cName)); err == nil {
			t.Fatal("actual C export overwrote a different explicit native import")
		}
	})
	t.Run("matching explicit dynamic import", func(t *testing.T) {
		ctx, pkg := consumer("package consumer\n//go:cgo_import_dynamic native_component_export native_component_export\n", first)
		if err := bindNativeAsmExports(ctx, pkg, pkg.Package, plan(firstExport.cName)); err != nil {
			t.Fatal(err)
		}
	})
	// A worker snapshot can omit an implementation not used by this native
	// component. It cannot grant the missing implementation a callable role.
	t.Run("unused and required missing implementation", func(t *testing.T) {
		snapshot := *first
		incomplete := firstExport
		incomplete.sig = nil
		snapshot.linkSnapshot = &packageLinkSnapshot{cExports: []cExport{incomplete}}
		ctx, pkg := consumer("package consumer\n//go:cgo_import_dynamic imported external\n", &snapshot)
		if err := bindNativeAsmExports(ctx, pkg, pkg.Package, plan("imported")); err != nil {
			t.Fatalf("unrelated missing export implementation blocked an explicit import: %v", err)
		}
		if err := bindNativeAsmExports(ctx, pkg, pkg.Package, plan(firstExport.cName)); err == nil {
			t.Fatal("required export with no actual compiled signature acquired a callable role")
		}
	})
}

func TestPackageCExportDeclarationsDirectEntry(t *testing.T) {
	for _, representation := range []string{"direct C entry", "qualified Go entry", "qualified entry wins", "missing implementation"} {
		t.Run(representation, func(t *testing.T) {
			prog := llssa.NewProgram(&llssa.Target{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH})
			t.Cleanup(prog.Dispose)
			pkg := prog.NewPackage("provider", "example.com/direct-export")
			signature := types.NewSignatureType(nil, nil, nil, nil, nil, false)
			const goName, cName = "example.com/direct-export.Native", "native_direct_export"
			var want *types.Signature
			if representation == "direct C entry" || representation == "qualified entry wins" {
				direct := pkg.NewFunc(cName, signature, llssa.CDecl)
				want = direct.RawType().(*types.Signature)
			}
			if representation == "qualified Go entry" || representation == "qualified entry wins" {
				results := types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Uintptr]))
				qualified := types.NewSignatureType(nil, nil, nil, nil, results, false)
				entry := pkg.NewFunc(goName, qualified, llssa.GoDecl)
				want = entry.RawType().(*types.Signature)
			}
			pkg.SetExport(goName, cName)
			exports := packageCExportDeclarations(pkg)
			if len(exports) != 1 || exports[0].goName != goName || exports[0].cName != cName || exports[0].sig != want {
				t.Fatalf("compiler export representation %q: got=%+v want actual signature=%v", representation, exports, want)
			}
		})
	}
}
