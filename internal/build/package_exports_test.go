//go:build !llgo

package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/exportdata"
	"github.com/xgo-dev/llgo/internal/packages"
)

func TestPackageExportsCacheValidation(t *testing.T) {
	t.Setenv(llgoBuildCache, "1")
	oldRoot := cacheRootFunc
	root := t.TempDir()
	cacheRootFunc = func() string { return root }
	defer func() { cacheRootFunc = oldRoot }()
	ctx := &context{conf: &packages.Config{}, buildConf: &Config{}, crossCompile: crosscompile.Export{LLVMTarget: "aarch64-apple-darwin"}}
	fset := token.NewFileSet()
	// An invalid annotation proves cache restoration does not collect source attributes.
	file, err := parser.ParseFile(fset, "dep.go", "package dep\n//llgo:cold invalid\nfunc Stop() {}", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg := &aPackage{Package: &packages.Package{ID: "dep", PkgPath: "dep", Name: "dep", Types: types.NewPackage("dep", "dep"), Fset: fset, Syntax: []*ast.File{file}}}
	ctx.pkgs = map[*packages.Package]Package{pkg.Package: pkg}
	m := newManifestBuilder()
	m.pkg.PkgPath = "dep"
	m.common.ExportVersion = exportdata.Version
	data, err := decodeManifest(m.Build())
	if err != nil {
		t.Fatal(err)
	}
	pkg.Fingerprint, err = manifestInputFingerprint(data)
	if err != nil {
		t.Fatal(err)
	}
	data.Exports = &exportdata.Package{Version: exportdata.Version, Functions: []exportdata.Function{{Name: "Stop", Cold: true, NoReturn: true}}}
	valid, err := buildManifestYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	paths := ctx.ensureCacheManager().PackagePaths(ctx.targetTriple(), pkg.PkgPath, pkg.Fingerprint)
	if err := ctx.ensureCacheManager().EnsureDir(paths); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Archive, []byte("archive"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, content string
		hit           bool
	}{
		{"valid", valid, true},
		{"missing", m.Build(), false},
		{"unsupported", strings.Replace(valid, "version: 1", "version: 99", 1), false},
		{"malformed", "exports: [", false},
		{"wrong-input", strings.Replace(valid, "pkg_path: dep", "pkg_path: other", 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "wrong-input" && tc.content == valid {
				t.Fatal("test did not change the package identity")
			}
			pkg.Exports = nil
			pkg.CacheHit = false
			if err := writeManifest(paths.Manifest, tc.content); err != nil {
				t.Fatal(err)
			}
			if hit := ctx.tryLoadFromCache(pkg); hit != tc.hit {
				t.Fatalf("cache hit = %v, want %v", hit, tc.hit)
			}
			if tc.hit {
				if err := ctx.preparePackageExports(); err != nil {
					t.Fatalf("restored records should bypass source collection: %v", err)
				}
				if got := pkg.Exports.Function("Stop"); !got.Cold || !got.NoReturn {
					t.Fatalf("lost cached attributes: %+v", got)
				}
			} else if err := ctx.preparePackageExports(); err == nil {
				t.Fatal("cache miss should collect and validate source attributes")
			}
		})
	}
	// An explicit empty record set is a valid cache result, not a request to scan source.
	data.Exports.Functions = nil
	empty, err := buildManifestYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeManifest(paths.Manifest, empty); err != nil {
		t.Fatal(err)
	}
	pkg.Exports = nil
	if !ctx.tryLoadFromCache(pkg) {
		t.Fatal("empty exports must be a cache hit")
	}
	if err := ctx.preparePackageExports(); err != nil {
		t.Fatal(err)
	}
}
