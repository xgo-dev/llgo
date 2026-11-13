package build

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestArchivePCLNCache(t *testing.T) {
	t.Setenv(llgoBuildCache, "1")
	root := t.TempDir()
	oldRoot := cacheRootFunc
	cacheRootFunc = func() string { return root }
	t.Cleanup(func() { cacheRootFunc = oldRoot })
	ctx := &context{
		buildConf:    &Config{PCLNMode: PCLNEmbedded},
		crossCompile: crosscompile.Export{LLVMTarget: "x86_64-unknown-linux-gnu"},
	}
	archive := filepath.Join(root, "source.a")
	if err := os.WriteFile(archive, []byte("archive fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	wantFuncs := []funcInfoRecord{
		{"p.Trace", "p.Trace", "p.go", 10, 2, 1},
		{"p.Trace.__llgo_fmv_avx2", "p.Trace", "p.go", 10, 2, 0},
	}
	wantLines := []pcLineRecord{{1<<63 + 7, "p.Trace.__llgo_fmv_avx2", "p.go", 14, 3}}
	m := newManifestBuilder()
	m.pkg.PkgPath = "p"
	pkg := &aPackage{
		Package:     &packages.Package{ID: "p", PkgPath: "p", Name: "p"},
		Fingerprint: "pcln-test", Manifest: m.Build(), ArchiveFile: archive,
		linkSnapshot: &packageLinkSnapshot{funcInfo: wantFuncs, pcLineInfo: wantLines},
	}
	if err := ctx.saveToCache(pkg); err != nil {
		t.Fatal(err)
	}
	paths := ctx.ensureCacheManager().PackagePaths(ctx.targetTriple(), pkg.PkgPath, pkg.Fingerprint)
	content, err := readManifest(paths.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	for _, mode := range []PCLNMode{PCLNEmbedded, PCLNExternal} {
		t.Run(mode.String(), func(t *testing.T) {
			ctx.buildConf.PCLNMode = mode
			// A reconstructed module without the physical variants must not
			// overwrite the cached function identity, flags or line IDs.
			loaded := &aPackage{Package: pkg.Package, Fingerprint: pkg.Fingerprint, LPkg: prog.NewPackage("p", "p")}
			if !ctx.tryLoadFromCache(loaded) {
				t.Fatal("valid archive metadata was not reused")
			}
			if got := collectFuncInfo([]Package{loaded}); !reflect.DeepEqual(got, wantFuncs) {
				t.Fatalf("cached functions = %#v, want %#v", got, wantFuncs)
			}
			if got := collectPCLineInfo([]Package{loaded}); !reflect.DeepEqual(got, wantLines) {
				t.Fatalf("cached lines = %#v, want %#v", got, wantLines)
			}
			ctx.snapshotBackendPackage(loaded)
			if !reflect.DeepEqual(loaded.linkSnapshot.funcInfo, wantFuncs) || !reflect.DeepEqual(loaded.linkSnapshot.pcLineInfo, wantLines) {
				t.Fatal("isolated backend snapshot lost archive metadata")
			}
			// If this package is rebuilt, its new module is authoritative:
			// neither the previous snapshot nor cached records may survive.
			loaded.CacheHit = false
			fresh := exportArchivePCLN(loaded)
			if len(fresh.Functions) != 0 || len(fresh.Lines) != 0 {
				t.Fatal("new backend capture reused stale archive records")
			}
			ctx.snapshotBackendPackage(loaded)
			if len(loaded.linkSnapshot.funcInfo) != 0 || len(loaded.linkSnapshot.pcLineInfo) != 0 {
				t.Fatal("new backend snapshot reused stale archive records")
			}
			if len(collectFuncInfo([]Package{loaded})) != 0 || len(collectPCLineInfo([]Package{loaded})) != 0 {
				t.Fatal("link collectors reused invalidated archive records")
			}
		})
	}
	for _, tc := range []struct {
		name string
		pcln *archivePCLNMetadata
	}{
		{"missing", nil},
		{"unknown-version", &archivePCLNMetadata{Version: archivePCLNVersion + 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := decodeManifest(content)
			if err != nil {
				t.Fatal(err)
			}
			data.Metadata.PCLN = tc.pcln
			changed, err := buildManifestYAML(data)
			if err != nil {
				t.Fatal(err)
			}
			if err := writeManifest(paths.Manifest, changed); err != nil {
				t.Fatal(err)
			}
			for _, mode := range []PCLNMode{PCLNEmbedded, PCLNExternal, PCLNNone} {
				ctx.buildConf.PCLNMode = mode
				loaded := &aPackage{Package: pkg.Package, Fingerprint: pkg.Fingerprint}
				if got, want := ctx.tryLoadFromCache(loaded), mode == PCLNNone; got != want {
					t.Fatalf("%s cache hit = %v, want %v", mode, got, want)
				}
			}
		})
	}
}
