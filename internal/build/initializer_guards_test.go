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

func TestOptionalInitializersMaps(t *testing.T) {
	if !buildenv.Dev {
		t.Skip("native size profile requires DCE")
	}
	const data = `package data
var value = &struct{ N int }{N: 17}
var Table = map[string]*struct{ N int }{"key":value}
func Lookup() int { return Table["key"].N }
`
	for _, tc := range []struct {
		name, source, entry string
		asmRoot             string
		open, linkname      bool
		wantGuard, wantDrop bool
	}{
		{name: "unused table", entry: `import "data"; func main() {}`, wantGuard: true, wantDrop: true},
		{name: "lookup", entry: `import "data"; func main() { println(data.Lookup()) }`, wantGuard: true},
		{name: "address", entry: `import "data"; func main() { println(&data.Table) }`, wantGuard: true},
		{name: "blank import", entry: `import _ "data"; func main() {}`, wantGuard: true},
		{name: "unknown foreign input", entry: `import "data"; func main() {}`, open: true, wantGuard: true},
		{name: "assembly global", entry: `import "data"; func main() {}`, asmRoot: "data.Table", wantGuard: true},
		{name: "assembly caller", entry: `import "data"; func main() {}`, asmRoot: "data.Lookup", wantGuard: true},
		{name: "linkname caller", entry: `import "data"; func main() {}`, linkname: true, wantGuard: true},
		{name: "explicit initialization", source: data + `func init() { println("effect") }`, entry: `import "data"; func main() {}`},
		{name: "interface key", source: `package data; var Table = map[any]int{"key":17}`, entry: `import "data"; func main() {}`},
		{name: "computed value", source: `package data; func n() int { return 17 }; var Table = map[string]int{"key":n()}`, entry: `import "data"; func main() {}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
			imports := make(codecInitTestImporter)
			goroot := t.TempDir()
			pkgs := make(map[string]*aPackage)
			add := func(path, source string) *packages.Package {
				pkg := checkFormatTestPackage(t, prog.Fset, path, source, types.Config{
					Importer: imports, DisableUnusedImportCheck: true,
				})
				pkg.ID = path
				pkg.Dir = filepath.Join(goroot, "src", path)
				imports[path] = pkg.Types
				pkgs[path] = &aPackage{Package: pkg}
				prog.CreatePackage(pkg.Types, pkg.Syntax, pkg.TypesInfo, true)
				return pkg
			}
			source := tc.source
			if source == "" {
				source = data
			}
			add("data", source)
			entry := add("main", "package main\n"+tc.entry)
			prog.Build()
			backend := llssa.NewProgram(&llssa.Target{GOOS: "darwin", GOARCH: "arm64"})
			defer backend.Dispose()
			ctx := &context{
				prog: backend, progSSA: prog, pkgByID: pkgs, initial: []*packages.Package{entry},
				nativeClosedProgram: !tc.open,
				buildConf: &Config{
					Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "darwin", Goarch: "arm64",
					LTO: lto.Full, OptLevel: optlevel.Oz, DeadcodeDrop: true, DisableGoGlobalDCE: true,
				},
			}
			if tc.asmRoot != "" {
				ctx.sizePruningRoots = []string{tc.asmRoot}
			}
			if tc.linkname {
				backend.SetLinkname("other.lookup", "data.Lookup")
			}
			configureOptionalInitializers(ctx, goroot, nil)
			if got := len(pkgs["data"].guardedInitializers) != 0; got != tc.wantGuard {
				t.Fatalf("archive guard = %v, want %v", got, tc.wantGuard)
			}
			var want []string
			if tc.wantDrop {
				want = []string{"_llgo_init_var.data.Table"}
			}
			if !slices.Equal(ctx.initializerGuardsOff, want) {
				t.Fatalf("link overrides = %v, want %v", ctx.initializerGuardsOff, want)
			}
		})
	}
}

func TestOptionalInitializersStandardFiles(t *testing.T) {
	const source = `package os
type File struct{}
func NewFile(uintptr, string) *File { return new(File) }
func other(uintptr, string) *File { return new(File) }
func filename() string { return "/dev/stderr" }
var Stdin = NewFile(0, "/dev/stdin")
var Stdout = NewFile(2, "/dev/stdout")
var Stderr = NewFile(2, filename())
var Other = NewFile(0, "/dev/stdin")
var OtherCall = other(1, "/dev/stdout")
`
	prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
	pkg := checkFormatTestPackage(t, prog.Fset, "os", source, types.Config{})
	ssapkg := prog.CreatePackage(pkg.Types, pkg.Syntax, pkg.TypesInfo, true)
	prog.Build()
	groups := optionalStandardFileInitializers(ssapkg.Func("init"))
	if len(groups) != 1 || groups[0].Global.Name() != "Stdin" {
		t.Fatalf("wanted exactly the matching standard file initializer, got %v", groups)
	}
	if groups := optionalStandardFileInitializers(ssapkg.Func("NewFile")); len(groups) != 0 {
		t.Fatalf("matched initialization in an ordinary function: %v", groups)
	}
}

func TestOptionalInitializersStaticSlices(t *testing.T) {
	if !buildenv.Dev {
		t.Skip("Wasm size profile requires DCE")
	}
	const data = `package data
var Element = &struct{ N int }{N: 17}
var Table = []*struct{ N int }{Element}
`
	for _, tt := range []struct {
		name, entry, assembly string
		linkAlias, wantDrop   bool
	}{
		{name: "unobserved", entry: "func main() {}", wantDrop: true},
		{name: "table read", entry: "func main() { println(data.Table[0].N) }"},
		{name: "other alias read", entry: "func main() { println(data.Element.N) }"},
		{name: "init read", entry: "var external = data.Element; func main() {}"},
		{name: "assembly alias", entry: "func main() {}", assembly: "data.Element"},
		{name: "linkname alias", entry: "func main() {}", linkAlias: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
			pkg := checkFormatTestPackage(t, prog.Fset, "data", data, types.Config{})
			entry := checkFormatTestPackage(t, prog.Fset, "main", "package main\nimport \"data\"\n"+tt.entry,
				types.Config{Importer: codecInitTestImporter{"data": pkg.Types}, DisableUnusedImportCheck: true})
			prog.CreatePackage(pkg.Types, pkg.Syntax, pkg.TypesInfo, true)
			prog.CreatePackage(entry.Types, entry.Syntax, entry.TypesInfo, true)
			prog.Build()
			root := t.TempDir()
			pkg.Dir = filepath.Join(root, "src", "data")
			backend := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
			defer backend.Dispose()
			if tt.linkAlias {
				backend.SetLinkname("other.element", "data.Element")
			}
			archive := &aPackage{Package: pkg}
			ctx := &context{
				prog: backend, progSSA: prog, initial: []*packages.Package{entry}, wasmClosedProgram: true,
				pkgByID:      map[string]*aPackage{"data": archive},
				crossCompile: crosscompile.Export{WasmProvider: crosscompile.WasmProviderWASI, WasmProfile: crosscompile.WasmProfileW32},
				buildConf: &Config{
					Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "wasip1", Goarch: "wasm", WasmOpt: WasmOptSize,
					LTO: lto.Full, OptLevel: optlevel.Oz, DeadcodeDrop: true, DisableGoGlobalDCE: true, Verbose: true,
				},
			}
			inputs := &linkPruningInputs{goroot: root}
			if tt.assembly != "" {
				inputs.roots = []string{tt.assembly}
			}
			configureOptionalInitializers(ctx, root, inputs)
			if len(archive.guardedInitializers) != 1 {
				t.Fatal("archive must expose the same slice guard for every entry")
			}
			var want []string
			if tt.wantDrop {
				want = []string{"_llgo_init_var.data.Table"}
			}
			if !slices.Equal(ctx.initializerGuardsOff, want) {
				t.Fatalf("disabled guards = %v, want %v", ctx.initializerGuardsOff, want)
			}
		})
	}
}

func TestOptionalInitializersBrowserBoundaries(t *testing.T) {
	if !buildenv.Dev {
		t.Skip("browser size profile requires DCE")
	}
	prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
	osPkg := checkFormatTestPackage(t, prog.Fset, "os", `package os
type File struct{}
func NewFile(uintptr, string) *File { return new(File) }
var Stdout = NewFile(1, "/dev/stdout")
`, types.Config{})
	entry := checkFormatTestPackage(t, prog.Fset, "main", "package main; func main() {}", types.Config{})
	prog.CreatePackage(osPkg.Types, osPkg.Syntax, osPkg.TypesInfo, true)
	prog.CreatePackage(entry.Types, entry.Syntax, entry.TypesInfo, true)
	prog.Build()

	goroot := t.TempDir()
	osPkg.Dir = filepath.Join(goroot, "src", "os")
	backend := llssa.NewProgram(&llssa.Target{GOOS: "js", GOARCH: "wasm"})
	defer backend.Dispose()
	for _, tc := range []struct {
		name    string
		inputs  *linkPruningInputs
		closed  bool
		wantOff bool
	}{
		{name: "closed", inputs: &linkPruningInputs{}, closed: true, wantOff: true},
		{name: "open", inputs: &linkPruningInputs{}},
		{name: "unknown inputs", closed: true},
		{name: "opaque assembly", inputs: &linkPruningInputs{opaqueAsmPkg: "unknown"}, closed: true},
		{name: "assembly reads stdout", inputs: &linkPruningInputs{roots: []string{"os.Stdout"}}, closed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			archive := &aPackage{Package: osPkg}
			ctx := &context{
				prog: backend, progSSA: prog, initial: []*packages.Package{entry},
				pkgByID: map[string]*aPackage{"os": archive}, wasmClosedProgram: tc.closed,
				crossCompile: crosscompile.Export{WasmProvider: crosscompile.WasmProviderEmscripten, WasmProfile: crosscompile.WasmProfileJ32},
				buildConf: &Config{
					Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm", WasmOpt: WasmOptSize,
					LTO: lto.Full, OptLevel: optlevel.Oz, DeadcodeDrop: true, DisableGoGlobalDCE: true,
				},
			}
			configureOptionalInitializers(ctx, goroot, tc.inputs)
			if len(archive.guardedInitializers) != 1 {
				t.Fatal("archive must provide the same initializer guard in every link")
			}
			var want []string
			if tc.wantOff {
				want = []string{"_llgo_init_var.os.Stdout"}
			}
			if !slices.Equal(ctx.initializerGuardsOff, want) {
				t.Fatalf("link guards disabled = %v, want %v", ctx.initializerGuardsOff, want)
			}
		})
	}
}
