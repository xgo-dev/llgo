package build

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/xgo-dev/llgo/internal/buildenv"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/meta"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func reflectCacheContext(t *testing.T) *context {
	t.Helper()
	if !buildenv.Dev || runtime.GOOS != "darwin" {
		t.Skip("development macOS experiment")
	}
	return &context{mode: ModeBuild, initial: []*packages.Package{{}},
		buildConf: &Config{DeadcodeDrop: true, BuildMode: BuildModeExe,
			Goos: runtime.GOOS, Goarch: runtime.GOARCH}}
}

func TestReflectIRPolicy(t *testing.T) {
	ctx := reflectCacheContext(t)
	for _, backend := range []struct {
		name string
		lto  lto.Mode
	}{
		{"strong", lto.Off},
		{"thin", lto.Thin},
	} {
		t.Run(backend.name, func(t *testing.T) {
			for _, debug := range []DebugArtifactMode{DebugArtifactDefault, DebugArtifactNone, DebugArtifactEmbedded, DebugArtifactExternal, DebugArtifactHost} {
				t.Run(debug.String(), func(t *testing.T) {
					conf := *ctx.buildConf
					conf.LTO, conf.DebugArtifactMode = backend.lto, debug
					other := *ctx
					other.buildConf = &conf
					want := backend.lto == lto.Off || debug == DebugArtifactNone
					if got := useDeadcodeReflectNames(&other); got != want {
						t.Errorf("reflection refinement = %v, want %v", got, want)
					}
				})
			}
			for name, mutate := range map[string]func(*Config){
				"no Drop": func(c *Config) { c.DeadcodeDrop = false },
				"FullLTO": func(c *Config) { c.LTO = lto.Full },
				"FullLTO Drop": func(c *Config) {
					c.LTO, c.DisableGoGlobalDCE = lto.Full, true
				},
				"shared":    func(c *Config) { c.BuildMode = BuildModeCShared },
				"target":    func(c *Config) { c.Target = "wasi" },
				"IR output": func(c *Config) { c.GenLL = true },
			} {
				t.Run(name, func(t *testing.T) {
					conf := *ctx.buildConf
					conf.LTO, conf.DebugArtifactMode = backend.lto, DebugArtifactNone
					mutate(&conf)
					other := *ctx
					other.buildConf = &conf
					if useDeadcodeReflectNames(&other) {
						t.Fatal("accepted unsupported reflection analysis configuration")
					}
				})
			}
		})
	}
}

func TestReflectIRCacheRoundTrip(t *testing.T) {
	for _, backend := range []struct {
		name string
		lto  lto.Mode
	}{
		{"strong", lto.Off},
		{"thin", lto.Thin},
	} {
		t.Run(backend.name, func(t *testing.T) {
			testReflectIRCacheRoundTrip(t, backend.lto)
		})
	}
}

func testReflectIRCacheRoundTrip(t *testing.T, backend lto.Mode) {
	t.Helper()
	ctx := reflectCacheContext(t)
	ctx.buildConf.LTO, ctx.buildConf.DebugArtifactMode = backend, DebugArtifactNone
	t.Setenv(llgoBuildCache, "on")
	dir := t.TempDir()
	oldRoot := cacheRootFunc
	cacheRootFunc = func() string { return dir }
	t.Cleanup(func() { cacheRootFunc = oldRoot })
	llssa.Initialize(llssa.InitAll)
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	pkg := &aPackage{Package: &packages.Package{Name: "hooks", PkgPath: "example.com/hooks"},
		LPkg: prog.NewPackage("example.com/hooks", "hooks"), Fingerprint: "reflect-ir-test"}
	mb := meta.NewBuilder()
	mb.AddReflectMethodSite(mb.Sym("example.com/hooks.Invoke"), mb.Sym("example.com/hooks.Invoke$reflectsite$1"))
	var err error
	pkg.Meta, err = mb.Build()
	if err != nil {
		t.Fatal(err)
	}
	ctx.pkgs = map[*packages.Package]*aPackage{pkg.Package: pkg}
	defer ctx.closePackageMetas()
	mod := pkg.LPkg.Module()
	g := llvm.AddGlobal(mod, mod.Context().Int32Type(), "snapshot")
	g.SetInitializer(llvm.ConstInt(g.GlobalValueType(), 1, false))
	if err := captureDeadcodeReflectIR(ctx, pkg); err != nil {
		t.Fatal(err)
	}
	temporary := pkg.reflectIRTemp
	if pkg.reflectIRFile == "" || temporary == "" {
		t.Fatal("optimized analysis snapshot was not captured")
	}
	// Simulate codegen changing the live module after the snapshot.
	g.SetInitializer(llvm.ConstInt(g.GlobalValueType(), 2, false))
	checkSnapshot := func(path string) {
		t.Helper()
		if err := visitDeadcodeReflectIR(path, func(m llvm.Module) error {
			if m.NamedGlobal("snapshot").Initializer().ZExtValue() != 1 {
				t.Error("snapshot changed after codegen")
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	checkSnapshot(pkg.reflectIRFile)
	pkg.ArchiveFile = filepath.Join(dir, "package.a")
	if err := os.WriteFile(pkg.ArchiveFile, []byte("unchanged package archive"), 0644); err != nil {
		t.Fatal(err)
	}
	manifest := newManifestBuilder()
	manifest.pkg.PkgPath = pkg.PkgPath
	pkg.Manifest = manifest.Build()
	if err := ctx.saveToCache(pkg); err != nil {
		t.Fatal(err)
	}
	cached := &aPackage{Package: pkg.Package, Fingerprint: pkg.Fingerprint}
	if !ctx.tryLoadFromCache(cached) {
		t.Fatal("analysis sidecar did not survive cache publication")
	}
	defer cached.Meta.Close()
	checkSnapshot(cached.reflectIRFile)
	data, err := os.ReadFile(cached.ArchiveFile)
	if err != nil || string(data) != "unchanged package archive" {
		t.Fatal("package archive changed", err)
	}
	ctx.closePackageMetas()
	if _, err := os.Stat(temporary); !os.IsNotExist(err) {
		t.Fatal("temporary IR leaked", err)
	}
	checkSnapshot(cached.reflectIRFile)
	if err := os.Remove(cached.reflectIRFile); err != nil {
		t.Fatal(err)
	}
	if ctx.tryLoadFromCache(&aPackage{Package: pkg.Package, Fingerprint: pkg.Fingerprint}) {
		t.Fatal("accepted cache entry without its required analysis sidecar")
	}
}
