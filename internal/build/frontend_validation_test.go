package build

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/frontcache"
	"github.com/xgo-dev/llgo/internal/packages"
	gopackages "golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

func TestFrontendExportValidation(t *testing.T) {
	f := frontendProgramFixture(t)
	pkg := f.owners[f.entries[0].pkg]
	pkg.Name = "dep"
	pkg.Fingerprint = "archive"
	pkg.linkSnapshot = &packageLinkSnapshot{}
	key := "source"
	f.keys[pkg.ID] = key
	record := &frontendExport{Version: frontendExportVersion, Input: key, Functions: []frontendFunction{{ID: "F"}}}
	f.records = map[string]*frontendExport{pkg.ID: record}
	if err := f.store(pkg); err != nil {
		t.Fatal(err)
	}
	if got := f.load(pkg); got == nil || got.Archive != pkg.Fingerprint {
		t.Fatal("complete export failed validation")
	}
	for _, tc := range []struct {
		name   string
		change func(*frontendExport)
	}{
		{"version", func(r *frontendExport) { r.Version++ }},
		{"input", func(r *frontendExport) { r.Input = "stale" }},
		{"archive", func(r *frontendExport) { r.Archive = "" }},
		{"link", func(r *frontendExport) { r.Link = nil }},
		{"functions", func(r *frontendExport) { r.Functions = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := *record
			r.Archive, r.Link = "archive", &frontendLinkExport{}
			tc.change(&r)
			writeFrontendRecord(t, f.path(key), &r)
			if f.load(pkg) != nil {
				t.Fatal("incomplete or stale export accepted")
			}
		})
	}
	if err := os.WriteFile(f.path(key), []byte(`{"Digest":"bad","Payload":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if f.load(pkg) != nil {
		t.Fatal("corrupt export accepted")
	}
	// An unwritable cache must surface the publication failure; it must not
	// make a certificate visible for an archive without its frontend record.
	if err := os.Mkdir(f.path(key)+".blocked", 0700); err != nil {
		t.Fatal(err)
	}
	f.ctx.cacheManager.root = f.path(key) + ".blocked"
	if err := os.WriteFile(filepath.Join(f.ctx.cacheManager.root, "frontend"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := f.store(pkg); err == nil {
		t.Fatal("failed export publication was hidden")
	}
}

func TestFrontendTypeExportRequiresCompleteCertificate(t *testing.T) {
	f := frontendProgramFixture(t)
	pkg := &packages.Package{ID: "example/dep", PkgPath: "example/dep", Name: "dep", Module: &gopackages.Module{}}
	l := &frontendTypeLoader{ctx: f.ctx, keys: map[string]string{pkg.ID: "source"}, selected: make(map[string]bool)}
	record := &frontendTypeExport{Version: frontendExportVersion, Input: "source", Frontend: "frontend", Archive: "archive",
		Package: &frontcache.PackageExport{ID: pkg.ID, Path: pkg.PkgPath, Name: pkg.Name}}
	writeFrontendRecord(t, l.path("source"), record)
	if l.load(pkg, token.NewFileSet()) {
		t.Fatal("type export without archive accepted")
	}
	archive := f.ctx.ensureCacheManager().PackagePaths(f.ctx.targetTriple(), pkg.PkgPath, record.Archive).Archive
	if err := os.MkdirAll(filepath.Dir(archive), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if l.load(pkg, token.NewFileSet()) {
		t.Fatal("type export without frontend companion accepted")
	}
	writeFrontendRecord(t, f.path(record.Frontend), &frontendExport{})
	if !l.load(pkg, token.NewFileSet()) || !l.selected[pkg.ID] || !pkg.Types.Complete() {
		t.Fatal("complete type export was not restored")
	}
	for _, tc := range []struct {
		name   string
		change func(*frontendTypeExport)
	}{
		{"version", func(r *frontendTypeExport) { r.Version++ }},
		{"input", func(r *frontendTypeExport) { r.Input = "stale" }},
		{"package", func(r *frontendTypeExport) { r.Package = nil }},
		{"archive", func(r *frontendTypeExport) { r.Archive = "" }},
		{"frontend", func(r *frontendTypeExport) { r.Frontend = "" }},
		{"type graph", func(r *frontendTypeExport) {
			copy := *r.Package
			copy.Objects = []frontcache.PackageObject{{Kind: "var", Object: frontcache.Object{Name: "X", Type: -1}}}
			r.Package = &copy
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := *record
			tc.change(&r)
			writeFrontendRecord(t, l.path("source"), &r)
			if l.load(pkg, token.NewFileSet()) {
				t.Fatal("invalid type export accepted")
			}
		})
	}
}

func TestFrontendInvalidRuntimeTypeBuildsSource(t *testing.T) {
	f := frontendProgramFixture(t)
	pkg := f.owners[f.entries[0].pkg]
	record := &frontendExport{RuntimeTypes: []int{-1}}
	pkg.frontendOpaque = record
	f.records = map[string]*frontendExport{pkg.ID: record}
	f.built[pkg.SSA] = false
	f.types = frontcache.NewTypes(nil, f.ctx.progSSA.Fset)
	f.ctx.callerTracking = cl.NewCallerTracking()
	if err := f.materialize(); err != nil {
		t.Fatal(err)
	}
	if !f.built[pkg.SSA] || pkg.frontendOpaque != nil || len(pkg.SSA.Func("main").Blocks) == 0 {
		t.Fatal("invalid runtime type retained an opaque package")
	}
}

func TestFrontendMissingDependencySourcePreventsReuse(t *testing.T) {
	f := frontendProgramFixture(t)
	root := f.owners[f.entries[0].pkg]
	filename := filepath.Join(t.TempDir(), "dep.go")
	dep := &packages.Package{ID: "dep", PkgPath: "dep", Name: "dep", GoFiles: []string{filename}, CompiledGoFiles: []string{filename}}
	root.Imports = map[string]*packages.Package{"dep": dep}
	f.keys, f.locals = make(map[string]string), make(map[string]string)
	l := &frontendTypeLoader{ctx: f.ctx, keys: make(map[string]string)}
	if _, err := f.inputKey(root); err == nil || l.key(root.Package, make(map[string]bool)) != "" {
		t.Fatal("missing dependency source acquired a reusable certificate")
	}
	if f.load(root) != nil || f.programKey(0) != "" {
		t.Fatal("missing dependency source retained package or program exports")
	}
	// The same missing disk file is valid when supplied by a source overlay.
	f.ctx.buildConf.Overlay[filename] = []byte("package dep")
	first, err := f.inputKey(root)
	before := l.key(root.Package, make(map[string]bool))
	if err != nil || first == "" || before == "" {
		t.Fatal("overlay source did not repair the dependency", err)
	}
	f.ctx.buildConf.Overlay[filename] = []byte("package dep;type Changed int")
	f.keys, f.locals, l.keys = make(map[string]string), make(map[string]string), make(map[string]string)
	second, err := f.inputKey(root)
	if err != nil || first == second || before == l.key(root.Package, make(map[string]bool)) {
		t.Fatal("dependency overlay did not invalidate its importers", err)
	}
	dep.Imports = map[string]*packages.Package{"root": root.Package}
	l.keys = make(map[string]string)
	if l.key(root.Package, make(map[string]bool)) != "" {
		t.Fatal("cyclic unchecked imports acquired a type certificate")
	}
}

func TestFrontendCacheLoadsWasmProgramCertificate(t *testing.T) {
	f := frontendProgramFixture(t)
	f.ctx.mode = ModeBuild
	t.Setenv(llgoBuildCache, "1")
	t.Setenv(llgoEscapeCache, "1")
	cache := f.ctx.newFrontendCache(f.entries, ssa.InstantiateGenerics)
	if cache == nil || cache.programInput == "" || cache.program != nil {
		t.Fatal("new Wasm context acquired a program certificate")
	}
	cache.captureProgram()
	fresh := f.ctx.newFrontendCache(f.entries, ssa.InstantiateGenerics)
	if fresh == nil || fresh.program == nil || fresh.programInput != cache.programInput {
		t.Fatal("unchanged Wasm context did not load its certificate")
	}
}

func TestFrontendTypeExportsKeepSourceDependentDeclarations(t *testing.T) {
	f := frontendProgramFixture(t)
	for _, tc := range []struct {
		name, source string
		want         bool
	}{
		{"plain", "var X int", true},
		{"generic function", "func F[T any](x T) T{return x}", false},
		{"generic type", "type T[P any] struct{X P}", false},
		{"generic alias", "type T[P any] = []P", false},
		{"environment rewrite", "//llgo:env LLGO_TEST\nvar X int", false},
		{"missing source", "var X int", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := parser.ParseFile(f.ctx.progSSA.Fset, "decl.go", "package dep\n"+tc.source, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			checked, err := new(types.Config).Check("example/dep", f.ctx.progSSA.Fset, []*ast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			pkg := &aPackage{Package: &packages.Package{ID: checked.Path(), PkgPath: checked.Path(), Name: checked.Name(), Types: checked, Syntax: []*ast.File{file}, Module: &gopackages.Module{}}}
			l := &frontendTypeLoader{ctx: f.ctx, keys: map[string]string{pkg.ID: "source"}, exports: make(map[string]*frontendTypeExport)}
			if tc.name == "missing source" {
				delete(l.keys, pkg.ID)
				pkg.CompiledGoFiles = []string{filepath.Join(t.TempDir(), "missing.go")}
			}
			l.capture(pkg, &frontendExport{}, frontcache.NewTypes(nil))
			if (l.exports[pkg.ID] != nil) != tc.want {
				t.Fatal("source-dependent declaration eligibility changed")
			}
		})
	}
}

func writeFrontendRecord(t *testing.T, filename string, record any) {
	t.Helper()
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(frontendEnvelope{Digest: frontendDigest(data), Payload: data})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, wire, 0600); err != nil {
		t.Fatal(err)
	}
}
