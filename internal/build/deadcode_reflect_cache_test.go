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
	if !useDeadcodeReflectNames(ctx) {
		t.Fatal("ordinary Drop excluded")
	}
	for name, mutate := range map[string]func(*Config){
		"no Drop":   func(c *Config) { c.DeadcodeDrop = false },
		"ThinLTO":   func(c *Config) { c.LTO = lto.Thin },
		"FullLTO":   func(c *Config) { c.LTO = lto.Full },
		"shared":    func(c *Config) { c.BuildMode = BuildModeCShared },
		"target":    func(c *Config) { c.Target = "wasi" },
		"IR output": func(c *Config) { c.GenLL = true },
	} {
		conf := *ctx.buildConf
		mutate(&conf)
		other := *ctx
		other.buildConf = &conf
		if useDeadcodeReflectNames(&other) {
			t.Errorf("accepted %s", name)
		}
	}
}

func TestReflectIRCacheRoundTrip(t *testing.T) {
	ctx := reflectCacheContext(t)
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
	pkg.ArchiveFile = filepath.Join(dir, "native.a")
	if err := os.WriteFile(pkg.ArchiveFile, []byte("native archive"), 0644); err != nil {
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
	if err != nil || string(data) != "native archive" {
		t.Fatal("native archive changed", err)
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
		t.Fatal("accepted native cache entry without its required analysis sidecar")
	}
}
