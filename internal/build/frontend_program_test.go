package build

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/types"
	"os"
	"reflect"
	"testing"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/goescape"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func frontendProgramFixture(t *testing.T) *frontendCache {
	t.Helper()
	const source = `package main;var Dead=map[int]int{1:2};func main(){println("ok")}`
	pkg := buildWasmReflectTestProgram(t, source)
	extraTypes := types.NewPackage("extra", "extra")
	extraTypes.MarkComplete()
	extra := pkg.Prog.CreatePackage(extraTypes, nil, nil, true)
	file, err := parser.ParseFile(pkg.Prog.Fset, "extra.go", "package extra", 0)
	if err != nil {
		t.Fatal(err)
	}
	root := &packages.Package{ID: pkg.Pkg.Path(), PkgPath: pkg.Pkg.Path(), Name: "main", Types: pkg.Pkg}
	ap := &aPackage{Package: root, SSA: pkg}
	prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProvider: "wasi"})
	t.Cleanup(prog.Dispose)
	ctx := &context{prog: prog, progSSA: pkg.Prog, conf: &packages.Config{}, mode: ModeGen, buildConf: &Config{Goarch: "wasm", BuildMode: BuildModeExe, Overlay: map[string][]byte{"extra.go": []byte("package extra")}}, llvmVersion: "test",
		initial: []*packages.Package{root}, pkgByID: map[string]*aPackage{root.ID: ap}, cacheManager: &cacheManager{root: t.TempDir()}}
	ctx.frontendOptions.UnusedGlobalInits = cl.FindUnusedGlobalInits(pkg.Prog, func(*ssa.Global) bool { return false })
	ctx.wasmClosedProgram = true
	f := &frontendCache{ctx: ctx, keys: make(map[string]string), locals: make(map[string]string), owners: map[*ssa.Package]Package{pkg: ap}, built: map[*ssa.Package]bool{pkg: true}, entries: []ssaBuildEntry{{id: root.ID, pkg: pkg}, {id: "extra", pkg: extra, syntax: []*ast.File{file}}}}
	ctx.frontend = f
	f.programInput = f.programKey(ssa.InstantiateGenerics)
	if f.programInput == "" {
		t.Fatal("empty program key")
	}
	return f
}

func TestFrontendProgramContextIdentity(t *testing.T) {
	f := frontendProgramFixture(t)
	want := f.programInput
	for _, tc := range []struct {
		name   string
		change func()
		undo   func()
	}{
		{"link flags", func() { f.ctx.buildConf.LinkOptions.ExternalLinkerFlags = "-lforeign" }, func() { f.ctx.buildConf.LinkOptions.ExternalLinkerFlags = "" }},
		{"asyncify", func() { f.ctx.crossCompile.WasmPostLink.Asyncify = true }, func() { f.ctx.crossCompile.WasmPostLink.Asyncify = false }},
		{"debugger frames", func() { f.ctx.frontendOptions.DebuggerFrames = true }, func() { f.ctx.frontendOptions.DebuggerFrames = false }},
		{"root identity", func() { f.ctx.initial[0].ID += ".test" }, func() { f.ctx.initial[0].ID = f.ctx.initial[0].PkgPath }},
		{"unowned source", func() { f.ctx.buildConf.Overlay["extra.go"] = []byte("package extra;var X int") }, func() { f.ctx.buildConf.Overlay["extra.go"] = []byte("package extra") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.change()
			if got := f.programKey(ssa.InstantiateGenerics); got == "" || got == want {
				t.Fatal("context change reused program key")
			}
			tc.undo()
			if got := f.programKey(ssa.InstantiateGenerics); got != want {
				t.Fatal("restored context changed identity")
			}
		})
	}
	if f.programKey(ssa.InstantiateGenerics|ssa.GlobalDebug) == want {
		t.Fatal("builder mode ignored")
	}
	t.Setenv("LDFLAGS", "-lforeign")
	if f.programKey(ssa.InstantiateGenerics) == want {
		t.Fatal("environment linker input ignored")
	}
	delete(f.ctx.buildConf.Overlay, "extra.go")
	if f.programKey(ssa.InstantiateGenerics) != "" {
		t.Fatal("missing source should prevent reuse")
	}
}

func TestFrontendProgramExportRestoresAndRejectsInvalidContexts(t *testing.T) {
	f := frontendProgramFixture(t)
	if f.loadProgram() != nil {
		t.Fatal("missing record accepted")
	}
	if f.restoreInitializers() == nil {
		t.Fatal("missing context accepted")
	}
	f.captureProgram()
	record := f.loadProgram()
	if record == nil {
		t.Fatal("published record not loaded")
	}
	before := f.ctx.frontendOptions.UnusedGlobalInits.Export(f.entries[0].pkg)
	f.program = record
	if err := f.restoreInitializers(); err != nil {
		t.Fatal(err)
	}
	if got := f.ctx.frontendOptions.UnusedGlobalInits.Export(f.entries[0].pkg); !reflect.DeepEqual(before, got) {
		t.Fatal("initializer plan changed")
	}
	f.built[f.entries[0].pkg] = false
	f.captureProgram()
	if got := f.loadProgram(); got == nil || !reflect.DeepEqual(got.Inits, record.Inits) {
		t.Fatal("opaque initializer plan lost")
	}
	original, err := os.ReadFile(f.programPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*frontendProgramExport)
	}{
		{"schema", func(r *frontendProgramExport) { r.Version++ }},
		{"input", func(r *frontendProgramExport) { r.Input = "different" }},
		{"roots", func(r *frontendProgramExport) { r.Initials = map[string]frontendProgramFeatures{} }},
		{"initializer inventory", func(r *frontendProgramExport) { r.Inits = map[string]cl.GlobalInitExport{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var envelope frontendEnvelope
			if err := json.Unmarshal(original, &envelope); err != nil {
				t.Fatal(err)
			}
			var r frontendProgramExport
			if err := json.Unmarshal(envelope.Payload, &r); err != nil {
				t.Fatal(err)
			}
			tc.mutate(&r)
			data, _ := json.Marshal(r)
			wire, _ := json.Marshal(frontendEnvelope{Digest: frontendDigest(data), Payload: data})
			if err := os.WriteFile(f.programPath(), wire, 0600); err != nil {
				t.Fatal(err)
			}
			if f.loadProgram() != nil {
				t.Fatal("incompatible record accepted")
			}
		})
	}
	for _, data := range [][]byte{[]byte("{"), []byte(`{"Digest":"bad","Payload":{}}`)} {
		if err := os.WriteFile(f.programPath(), data, 0600); err != nil {
			t.Fatal(err)
		}
		if f.loadProgram() != nil {
			t.Fatal("corrupt record accepted")
		}
	}
	big, err := os.Create(f.programPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := big.Truncate(goescape.MaxCacheBytes + 1); err != nil {
		t.Fatal(err)
	}
	big.Close()
	if f.loadProgram() != nil {
		t.Fatal("oversized export accepted")
	}
	f.programInput = ""
	if f.loadProgram() != nil {
		t.Fatal("empty key accepted")
	}
	f.captureProgram()
}

func TestFrontendProgramInvalidInitializerFallsBack(t *testing.T) {
	f := frontendProgramFixture(t)
	f.captureProgram()
	f.program = f.loadProgram()
	configureUnusedGlobalInits(f.ctx)
	if f.program == nil {
		t.Fatal("valid initializer plan was discarded")
	}
	plan := f.program.Inits[f.entries[0].id]
	if len(plan.Instructions) == 0 {
		t.Fatal("fixture has no removals")
	}
	plan.Instructions[0].Text = "changed"
	f.program.Inits[f.entries[0].id] = plan
	if err := f.restoreInitializers(); err == nil {
		t.Fatal("changed initializer accepted")
	}
	// Opaque link metadata cannot authorize source instruction changes.
	configureUnusedGlobalInits(f.ctx)
	if f.program != nil || f.ctx.frontendOptions.UnusedGlobalInits != nil {
		t.Fatal("invalid initializer context survived fallback")
	}
	for _, entry := range f.entries {
		if !f.built[entry.pkg] {
			t.Fatal("fallback left a package without source SSA")
		}
	}
}

func TestFrontendProgramRestoresWasmFeatureDecisions(t *testing.T) {
	f := frontendProgramFixture(t)
	target := f.ctx.prog.Target()
	target.WasmReflectBridges, target.WasmFuncInfoEntries = true, true
	f.ctx.wasmRuntimeFree = true
	f.captureProgram()
	f.program = f.loadProgram()
	if f.program == nil {
		t.Fatal("missing program export")
	}
	// Start with the opposite defaults to check that the deserialized program
	// contract controls Wasm features even when dependency bodies are absent.
	target.WasmReflectBridges, target.WasmFuncInfoEntries = false, false
	f.ctx.wasmRuntimeFree = false
	f.ctx.frontendOptions.ShadowStack = true
	configureWasmReflectBridges(f.ctx)
	configureWasmFuncInfoEntries(f.ctx)
	configureWasmRuntimeFree(f.ctx)
	if !target.WasmReflectBridges || !target.WasmFuncInfoEntries || !f.ctx.wasmRuntimeFree || f.ctx.frontendOptions.ShadowStack {
		t.Fatal("cached Wasm features changed")
	}
	// Multiple roots retain their separate feature contracts when grouped.
	f.ctx.mode = ModeBuild
	extra := &packages.Package{ID: f.entries[1].id, Types: f.entries[1].pkg.Pkg}
	f.ctx.initial = append(f.ctx.initial, extra)
	f.program.Initials[f.ctx.initial[0].ID] = frontendProgramFeatures{ReflectBridges: true}
	f.program.Initials[extra.ID] = frontendProgramFeatures{FuncInfoEntries: true}
	groups := groupInitialBuilds(f.ctx, nil)
	if len(groups) != 2 || !groups[0].features.reflectBridges || groups[0].features.funcInfoEntries || groups[1].features.reflectBridges || !groups[1].features.funcInfoEntries {
		t.Fatal("per-root Wasm contracts were merged")
	}
}
