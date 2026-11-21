package build

import (
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xgo-dev/llgo/internal/buildenv"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestOptionalInitializersLiteralValues(t *testing.T) {
	if !buildenv.Dev {
		t.Skip("Wasm size profile requires DCE")
	}
	const onceValue = `func OnceValue[T any](f func() T) func() T {
	d := struct { f func() T; result T }{f: f}
	return func() T { d.result = d.f(); return d.result }
}`
	const makeCallback = "sync.OnceValue(func() int { return tables.Values[0] })"
	for _, tt := range []struct {
		name, constructor, rhs, entry, syncSource, alias string
		hasTable, wantGuard, wantDrop, wantTableDrop     bool
	}{
		{name: "unused", wantGuard: true, wantDrop: true},
		{name: "used", entry: "println(data.Sentinel.Error())", wantGuard: true},
		{name: "constructor alias", alias: "errors.New", wantGuard: true},
		{name: "constructor with side effects", constructor: "println(s); return &message{s}"},
		{name: "constructor returns nil", constructor: "return nil"},
		{name: "dynamic input", rhs: "errors.New(text())"},
		{name: "constant interface", rhs: "any(struct{ N int }{})", wantGuard: true, wantDrop: true},
		{name: "live constant interface", rhs: "any(struct{ N int }{})", entry: "println(data.Sentinel != nil)", wantGuard: true},
		{
			name:  "unused callback cannot retain static data through an indirect call",
			rhs:   "struct{ New func() }{func() { println(tables.Values[0]) }}",
			entry: "invoke(func() {})", hasTable: true, wantGuard: true, wantDrop: true, wantTableDrop: true,
		},
		{
			name: "live callback retains its static data", rhs: "struct{ New func() }{func() { println(tables.Values[0]) }}",
			entry: "invoke(data.Sentinel.New)", hasTable: true, wantGuard: true,
		},
		{
			name: "initializer calls a function", rhs: "struct{ New string }{text()}",
		},
		{
			name: "unused callback factory cannot retain static data", rhs: makeCallback, syncSource: onceValue,
			entry: "invokeValue(func() int { return 0 })", hasTable: true, wantGuard: true, wantDrop: true, wantTableDrop: true,
		},
		{
			name: "live callback factory", rhs: makeCallback, syncSource: onceValue,
			entry: "invokeValue(data.Sentinel)", hasTable: true, wantGuard: true,
		},
		{
			name: "callback factory alias", rhs: makeCallback, syncSource: onceValue,
			alias: "sync.OnceValue[int]", entry: "invokeValue(func() int { return 0 })", hasTable: true, wantGuard: true,
		},
		{
			name: "effectful callback factory", rhs: makeCallback, hasTable: true,
			syncSource: `func OnceValue[T any](f func() T) func() T { println("effect"); return f }`,
			entry:      "invokeValue(func() int { return 0 })",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions|ssa.InstantiateGenerics)
			root := t.TempDir()
			pkgs := make(map[string]*aPackage)
			imports := make(codecInitTestImporter)
			add := func(path, source string) *packages.Package {
				pkg := checkFormatTestPackage(t, prog.Fset, path, source, types.Config{
					Importer: imports, DisableUnusedImportCheck: true,
				})
				pkg.Dir = filepath.Join(root, "src", path)
				pkgs[path] = &aPackage{Package: pkg}
				imports[path] = pkg.Types
				prog.CreatePackage(pkg.Types, pkg.Syntax, pkg.TypesInfo, true)
				return pkg
			}
			constructor := tt.constructor
			if constructor == "" {
				constructor = "return &message{s}"
			}
			add("errors", "package errors\ntype message struct { text string }\n"+
				"func (m *message) Error() string { return m.text }\n"+
				"func New(s string) error { "+constructor+" }")
			rhs := tt.rhs
			if rhs == "" {
				rhs = `errors.New("sentinel")`
			}
			dataImports := "import \"errors\"\n"
			if tt.syncSource != "" {
				add("sync", "package sync\n"+tt.syncSource)
				dataImports += "import \"sync\"\n"
			}
			if tt.hasTable {
				add("tables", "package tables\nvar Values = []int{42}")
				dataImports += "import \"tables\"\n"
			}
			add("data", "package data\n"+dataImports+"var Sentinel = "+rhs+"\nfunc text() string { return \"text\" }")
			entry := add("main", `package main
import "data"
func invoke(f func()) { f() }
func invokeValue(f func() int) int { return f() }
func main() { `+tt.entry+` }`)
			prog.Build()
			backend := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
			defer backend.Dispose()
			if tt.alias != "" {
				backend.SetLinkname("other.constructor", tt.alias)
			}
			ctx := &context{
				prog: backend, progSSA: prog, initial: []*packages.Package{entry},
				pkgByID: pkgs, wasmClosedProgram: true,
				crossCompile: crosscompile.Export{WasmProvider: crosscompile.WasmProviderWASI, WasmProfile: crosscompile.WasmProfileW32},
				buildConf: &Config{
					Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "wasip1", Goarch: "wasm", WasmOpt: WasmOptSize,
					LTO: lto.Full, OptLevel: optlevel.Oz, DeadcodeDrop: true, DisableGoGlobalDCE: true, Verbose: true,
				},
			}
			configureOptionalInitializers(ctx, root, &linkPruningInputs{goroot: root})
			if got := len(pkgs["data"].guardedInitializers) != 0; got != tt.wantGuard {
				t.Fatalf("archive guard = %v, want %v", got, tt.wantGuard)
			}
			var want []string
			if tt.wantDrop {
				want = []string{"_llgo_init_var.data.Sentinel"}
			}
			if tt.wantTableDrop {
				want = append(want, "_llgo_init_var.tables.Values")
			}
			if !slices.Equal(ctx.initializerGuardsOff, want) {
				t.Fatalf("disabled guards = %v, want %v", ctx.initializerGuardsOff, want)
			}
		})
	}
}
