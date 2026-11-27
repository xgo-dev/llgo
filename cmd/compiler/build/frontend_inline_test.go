package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/cmd/cl"
	"github.com/xgo-dev/llgo/cmd/compiler/frontcache"
	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	llpackages "github.com/xgo-dev/llgo/cmd/compiler/packages"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssautil"
)

func TestFrontendInlineReplacementBodies(t *testing.T) {
	t.Setenv(llgoBuildCache, "1")
	t.Setenv(llgoEscapeCache, "1")
	for _, test := range []struct {
		name, original, replacement, call string
		wantBuild, disableImports         bool
	}{
		{"function", "func Value(x T) int { for x > 0 { x-- }; return int(x) }", "func Value(x T) int { return int(x) + 1 }", "dep.Value(1)", true, false},
		{"value method", "func (x T) Value() int { for x > 0 { x-- }; return int(x) }", "func (x T) Value() int { return int(x) + 1 }", "dep.T(1).Value()", true, false},
		{"pointer method", "func (x *T) Value() int { for *x > 0 { *x-- }; return int(*x) }", "func (x *T) Value() int { return int(*x) + 1 }", "new(dep.T).Value()", true, false},
		{"bodyless method", "func (T) Value() int", "func (x T) Value() int { return int(x) + 1 }", "dep.T(1).Value()", true, false},
		{"noinline method", "//go:noinline\nfunc (x T) Value() int { return int(x) }", "func (x T) Value() int { return int(x) + 1 }", "dep.T(1).Value()", false, false},
		{"missing method", "func (x T) Value() int { return int(x) }", "", "dep.T(1).Value()", false, false},
		{"deferred method", "func (x T) Value() int { return int(x) }", "func (x T) Value() int { return int(x) + 1 }", "dep.T(1).Value()", false, true},
		{"deferred function", "func Value(x T) int { return int(x) }", "func Value(x T) int { return int(x) + 1 }", "dep.Value(1)", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, source := range map[string]string{
				"go.mod":     "module example.com/inline\n\ngo 1.24\n",
				"dep/dep.go": "package dep\ntype T int\n" + test.original,
				"alt/alt.go": "package dep\ntype T int\n" + test.replacement,
				"main.go":    "package main\nimport \"example.com/inline/dep\"\nfunc main() { println(" + test.call + ") }",
			} {
				if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(name)), 0755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(dir, name), source)
			}
			loaded, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax, Dir: dir}, "./...")
			if err != nil || packages.PrintErrors(loaded) != 0 {
				t.Fatalf("load fixture: %v", err)
			}
			cold, _ := packages.Packages(loaded, ssa.SanityCheckFunctions)
			cold.Build()
			tracking := cl.NewCallerTracking()
			tracking.Precompute(cold.AllPackages())
			bodyBudget, _, _ := cl.FrontendInlinePolicy()
			index := frontcache.NewTypes(nil, cold.Fset)
			records := make(map[string]*frontendExport)
			for fn := range ssautil.AllFunctions(cold) {
				if fn.Pkg == nil {
					continue
				}
				id := fn.Pkg.Pkg.Path()
				if records[id] == nil {
					records[id] = new(frontendExport)
				}
				facts := tracking.ExportFrontendFacts(fn.Pkg, fn)
				if test.wantBuild && fn.Name() == "Value" && (facts.InlineCost <= bodyBudget) != (id == "example.com/inline/alt") {
					t.Fatalf("unexpected inline cost for %s: %d", fn, facts.InlineCost)
				}
				records[id].Functions = append(records[id].Functions, frontendFunction{
					ID: frontcache.FunctionID(fn, index), Facts: facts,
				})
			}

			// Register fresh dependency SSA without bodies, as on a frontend
			// cache hit. Only the caller is built before inline preloading.
			warm, pkgs := packages.Packages(loaded, ssa.SanityCheckFunctions)
			ctx := &context{progSSA: warm, buildConf: NewDefaultConf(ModeBuild),
				callerTracking: cl.NewCallerTracking(), pkgByID: make(map[string]*aPackage)}
			ctx.frontendOptions.DisableInlineBodyImports = test.disableImports
			var entries []ssaBuildEntry
			for i, pkg := range pkgs {
				ctx.pkgByID[loaded[i].ID] = &aPackage{Package: loaded[i], SSA: pkg, CacheHit: true}
				entries = append(entries, ssaBuildEntry{id: loaded[i].ID, pkg: pkg, syntax: loaded[i].Syntax})
			}
			root, dep, alt := ctx.pkgByID["example.com/inline"], ctx.pkgByID["example.com/inline/dep"], ctx.pkgByID["example.com/inline/alt"]
			dep.AltPkg = &llpackages.Cached{Package: alt.Package, Types: alt.Types}
			delete(ctx.pkgByID, alt.ID)
			ctx.patches = cl.Patches{dep.PkgPath: {Alt: alt.SSA, Types: alt.Types}}
			root.CacheHit = false
			root.SSA.Build()
			f := ctx.newFrontendCache(entries, ssa.SanityCheckFunctions)
			if f == nil {
				t.Fatal("frontend cache disabled")
			}
			f.records, f.types = records, index
			f.built[root.SSA] = true
			for fn := range ssautil.AllFunctions(warm) {
				if fn.Pkg == alt.SSA && len(fn.Blocks) != 0 {
					t.Fatalf("replacement already has a body: %s", fn)
				}
			}
			f.bindFunctions()
			if err := f.importBodies(); err != nil {
				t.Fatal(err)
			}
			if got := f.built[alt.SSA]; got != test.wantBuild {
				t.Fatalf("replacement built = %v, want %v", got, test.wantBuild)
			}
			if test.wantBuild {
				hasBody := false
				for fn := range ssautil.AllFunctions(warm) {
					if fn.Pkg == alt.SSA && fn.Name() == "Value" {
						hasBody = len(fn.Blocks) != 0
					}
				}
				if !hasBody {
					t.Fatal("replacement body missing after preload")
				}
			}
		})
	}
}
