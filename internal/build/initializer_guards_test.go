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
	const mixed = `package data
var value = &struct{ N int }{N: 17}
var First = []*struct{ N int }{value}
var Table = map[string]*struct{ N int }{"key":value}
var Second = []*struct{ N int }{value}
func Lookup() int { return value.N }
`
	for _, tc := range []struct {
		name, source, entry     string
		asmRoot                 string
		open, linkname          bool
		embedded, wasi, browser bool
		wantGuard, wantDrop     bool
		wantOrder               []string
	}{
		{name: "unused table", entry: `import "data"; func main() {}`, wantGuard: true, wantDrop: true},
		{name: "lookup", entry: `import "data"; func main() { println(data.Lookup()) }`, wantGuard: true},
		{name: "address", entry: `import "data"; func main() { println(&data.Table) }`, wantGuard: true},
		{name: "blank import", entry: `import _ "data"; func main() {}`, wantGuard: true},
		{name: "unknown foreign input", entry: `import "data"; func main() {}`, open: true, wantGuard: true},
		{name: "assembly global", entry: `import "data"; func main() {}`, asmRoot: "data.Table", wantGuard: true},
		{name: "assembly caller", entry: `import "data"; func main() {}`, asmRoot: "data.Lookup", wantGuard: true},
		{name: "linkname caller", entry: `import "data"; func main() {}`, linkname: true, wantGuard: true},
		{name: "independent initialization", source: data + `func init() { println("effect") }`, entry: `import "data"; func main() {}`, wantGuard: true, wantDrop: true},
		{name: "native lookup during init", source: data + `func init() { println(Lookup()) }`, entry: `import "data"; func main() {}`, wantGuard: true},
		{name: "interface key", source: `package data; var Table = map[any]int{"key":17}`, entry: `import "data"; func main() {}`},
		{name: "computed value", source: `package data; func n() int { return 17 }; var Table = map[string]int{"key":n()}`, entry: `import "data"; func main() {}`},
		{name: "observable slice backing data keeps slices only", source: mixed, entry: `import "data"; func main() { println(data.Lookup()) }`, wantGuard: true, wantDrop: true, wantOrder: []string{"First", "Table", "Second"}},
		{name: "closed firmware", embedded: true, entry: `import "data"; func main() {}`, wantGuard: true, wantDrop: true},
		{name: "firmware with independent init", embedded: true, source: data + `func init() { println("effect") }`, entry: `import "data"; func main() {}`, wantGuard: true, wantDrop: true},
		{name: "firmware lookup during init", embedded: true, source: data + `func init() { println(Lookup()) }`, entry: `import "data"; func main() {}`, wantGuard: true},
		{name: "open firmware", embedded: true, open: true, entry: `import "data"; func main() {}`, wantGuard: true},
		{name: "closed WASI with independent init", wasi: true, source: data + `func init() { println("effect") }`, entry: `import "data"; func main() {}`, wantGuard: true, wantDrop: true},
		{name: "WASI lookup during init", wasi: true, source: data + `func init() { println(Lookup()) }`, entry: `import "data"; func main() {}`, wantGuard: true},
		{name: "WASI assembly root", wasi: true, entry: `import "data"; func main() {}`, asmRoot: "data.Table", wantGuard: true},
		{name: "open WASI", wasi: true, open: true, entry: `import "data"; func main() {}`, wantGuard: true},
		{name: "unused browser map", browser: true, entry: `import "data"; func main() {}`, wantGuard: true, wantDrop: true},
		{name: "browser with independent init", browser: true, source: data + `func init() { println("effect") }`, entry: `import "data"; func main() {}`, wantGuard: true, wantDrop: true},
		{name: "browser lookup during init", browser: true, source: data + `func init() { println(Lookup()) }`, entry: `import "data"; func main() {}`, wantGuard: true},
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
			var inputs *linkPruningInputs
			if tc.embedded {
				ctx.buildConf.Target = "cortex-m-mps2"
				ctx.buildConf.Goos = "linux"
				ctx.buildConf.Goarch = "arm"
				ctx.crossCompile.ExtraFiles = embeddedBootFiles(ctx.buildConf.Target)
				ctx.nativeClosedProgram = false
				ctx.embeddedClosedProgram = !tc.open
				inputs = &linkPruningInputs{goroot: goroot}
			}
			if tc.wasi || tc.browser {
				ctx.buildConf.Goos, ctx.buildConf.Goarch = "js", "wasm"
				ctx.buildConf.WasmOpt = WasmOptSize
				ctx.crossCompile.WasmProvider = crosscompile.WasmProviderEmscripten
				ctx.crossCompile.WasmProfile = crosscompile.WasmProfileJ32
				if tc.wasi {
					ctx.buildConf.Goos = "wasip1"
					ctx.crossCompile.WasmProvider = crosscompile.WasmProviderWASI
					ctx.crossCompile.WasmProfile = crosscompile.WasmProfileW32
				}
				ctx.nativeClosedProgram = false
				ctx.wasmClosedProgram = !tc.open
				inputs = &linkPruningInputs{goroot: goroot, roots: ctx.sizePruningRoots}
			}
			configureOptionalInitializers(ctx, goroot, inputs)
			if got := len(pkgs["data"].guardedInitializers) != 0; got != tc.wantGuard {
				t.Fatalf("archive guard = %v, want %v", got, tc.wantGuard)
			}
			if tc.wantOrder != nil {
				var order []string
				for _, block := range prog.Package(pkgs["data"].Types).Func("init").Blocks {
					for _, group := range pkgs["data"].guardedInitializers[block] {
						order = append(order, group.Global.Name())
					}
				}
				if !slices.Equal(order, tc.wantOrder) {
					t.Fatalf("guard order = %v, want %v", order, tc.wantOrder)
				}
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
	goroot := t.TempDir()
	pkg.Dir = filepath.Join(goroot, "src", "os")
	archive := &aPackage{Package: pkg}
	ctx := &context{
		progSSA: prog, pkgByID: map[string]*aPackage{"os": archive}, wasmClosedProgram: true,
		crossCompile: crosscompile.Export{WasmProvider: crosscompile.WasmProviderWASI, WasmProfile: crosscompile.WasmProfileW32},
		buildConf: &Config{
			Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "wasip1", Goarch: "wasm", WasmOpt: WasmOptSize,
			LTO: lto.Full, OptLevel: optlevel.Oz, DeadcodeDrop: true, DisableGoGlobalDCE: true,
		},
	}
	configureOptionalInitializers(ctx, goroot, &linkPruningInputs{})
	if len(archive.guardedInitializers) != 0 {
		t.Fatal("WASI requires the full standard descriptor initializers")
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

func TestOptionalInitializersWasmBoundaries(t *testing.T) {
	if !buildenv.Dev {
		t.Skip("browser size profile requires DCE")
	}
	prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
	osPkg := checkFormatTestPackage(t, prog.Fset, "os", `package os
type File struct{}
func NewFile(uintptr, string) *File { return new(File) }
var Stdin = NewFile(0, "/dev/stdin")
var Stdout = NewFile(1, "/dev/stdout")
var Stderr = NewFile(2, "/dev/stderr")
`, types.Config{})
	entry := checkFormatTestPackage(t, prog.Fset, "main", "package main; func main() {}", types.Config{})
	prog.CreatePackage(osPkg.Types, osPkg.Syntax, osPkg.TypesInfo, true)
	prog.CreatePackage(entry.Types, entry.Syntax, entry.TypesInfo, true)
	prog.Build()

	goroot := t.TempDir()
	osPkg.Dir = filepath.Join(goroot, "src", "os")
	backend := llssa.NewProgram(&llssa.Target{GOOS: "js", GOARCH: "wasm"})
	defer backend.Dispose()
	all := []string{"_llgo_init_var.os.Stderr", "_llgo_init_var.os.Stdin", "_llgo_init_var.os.Stdout"}
	for _, tc := range []struct {
		name    string
		inputs  *linkPruningInputs
		closed  bool
		wasi    bool
		wantOff []string
	}{
		{name: "closed", inputs: &linkPruningInputs{}, closed: true, wantOff: all},
		{name: "open", inputs: &linkPruningInputs{}},
		{name: "unknown inputs", closed: true},
		{name: "opaque assembly", inputs: &linkPruningInputs{opaqueAsmPkg: "unknown"}, closed: true},
		{name: "assembly reads stdout", inputs: &linkPruningInputs{roots: []string{"os.Stdout"}}, closed: true, wantOff: all[:2]},
		{name: "wasi closed", inputs: &linkPruningInputs{}, closed: true, wasi: true, wantOff: all},
		{name: "wasi open", inputs: &linkPruningInputs{}, wasi: true},
		{name: "wasi assembly reads stdout", inputs: &linkPruningInputs{roots: []string{"os.Stdout"}}, closed: true, wasi: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			archive := &aPackage{Package: osPkg}
			goos := "js"
			profile := crosscompile.Export{WasmProvider: crosscompile.WasmProviderEmscripten, WasmProfile: crosscompile.WasmProfileJ32}
			if tc.wasi {
				goos = "wasip1"
				profile = crosscompile.Export{WasmProvider: crosscompile.WasmProviderWASI, WasmProfile: crosscompile.WasmProfileW32}
			}
			ctx := &context{
				prog: backend, progSSA: prog, initial: []*packages.Package{entry},
				pkgByID: map[string]*aPackage{"os": archive}, wasmClosedProgram: tc.closed,
				crossCompile: profile,
				buildConf: &Config{
					Mode: ModeBuild, BuildMode: BuildModeExe, Goos: goos, Goarch: "wasm", WasmOpt: WasmOptSize,
					LTO: lto.Full, OptLevel: optlevel.Oz, DeadcodeDrop: true, DisableGoGlobalDCE: true,
				},
			}
			configureOptionalInitializers(ctx, goroot, tc.inputs)
			if len(archive.guardedInitializers) != 1 {
				t.Fatal("archive must provide the same initializer guard in every link")
			}
			if !slices.Equal(ctx.initializerGuardsOff, tc.wantOff) {
				t.Fatalf("link guards disabled = %v, want %v", ctx.initializerGuardsOff, tc.wantOff)
			}
		})
	}
}

func TestOptionalInitializersStdout(t *testing.T) {
	if !buildenv.Dev {
		t.Skip("stdout specialization requires DCE")
	}
	for _, tc := range []struct {
		name, body        string
		assembly          []string
		reflectiveMethods bool
		embedded          bool
		native            bool
		writer            string
		compact           bool
		rawAlias          bool
		externalFlags     bool
	}{
		{name: "literal", body: `fmt.Printf("hello")`, writer: "llgoWriteStdoutDescriptor"},
		{name: "deferred", body: `defer fmt.Printf("hello")`, writer: "llgoWriteStdoutDescriptor"},
		{name: "file writer", body: `os.WriteFile(); fmt.Printf("hello")`, writer: "llgoWriteClosedStdout"},
		{name: "assembly file writer", body: `fmt.Printf("hello")`, assembly: []string{pollWriteSymbol}, writer: "llgoWriteClosedStdout"},
		{name: "reflective file writer", body: `fmt.Printf("hello")`, reflectiveMethods: true, writer: "llgoWriteClosedStdout"},
		{name: "redirected", body: `os.Stdout = nil; fmt.Printf("hello")`},
		{name: "other descriptor", body: `println(os.Stderr); fmt.Printf("hello")`},
		{name: "assembly reference", body: `fmt.Printf("hello")`, assembly: []string{"os.Stdout"}},
		{name: "no literal calls", body: `fmt.Printf(os.Dynamic())`},
		{name: "embedded literal", embedded: true, body: `fmt.Printf("hello")`, writer: "llgoWriteStdoutDescriptor", compact: true},
		{name: "embedded observed result", embedded: true, body: `n, err := fmt.Printf("hello"); if err != nil || n != 5 { panic("failed") }`, writer: "llgoWriteStdoutDescriptor"},
		{name: "embedded raw alias", embedded: true, rawAlias: true, body: `fmt.Printf("hello")`},
		{name: "embedded linker flags", embedded: true, externalFlags: true, body: `fmt.Printf("hello")`, writer: "llgoWriteStdoutDescriptor"},
		{name: "embedded descriptor read", embedded: true, body: `println(syscall.Stdout); fmt.Printf("hello")`, writer: "llgoWriteStdoutDescriptor"},
		{name: "embedded file writer", embedded: true, body: `os.WriteFile(); fmt.Printf("hello")`, writer: "llgoWriteClosedStdout"},
		{name: "embedded descriptor write", embedded: true, body: `syscall.Stdout = 2; fmt.Printf("hello")`},
		{name: "embedded descriptor escape", embedded: true, body: `fmt.Printf("hello"); save(&syscall.Stdout)`},
		{name: "embedded descriptor symbol", embedded: true, body: `fmt.Printf("hello")`, assembly: []string{"syscall.Stdout"}},
		{name: "native literal", native: true, body: `fmt.Printf("hello")`, writer: "llgoWriteStdoutDescriptor"},
		{name: "native file writer", native: true, body: `os.WriteFile(); fmt.Printf("hello")`},
		{name: "native assembly file writer", native: true, body: `fmt.Printf("hello")`, assembly: []string{pollWriteSymbol}},
		{name: "native descriptor escape", native: true, body: `fmt.Printf("hello"); save(&syscall.Stdout)`},
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
				pkg.Dir = filepath.Join(goroot, "src", path)
				imports[path] = pkg.Types
				pkgs[path] = &aPackage{Package: pkg}
				prog.CreatePackage(pkg.Types, pkg.Syntax, pkg.TypesInfo, true)
				return pkg
			}
			add("internal/poll", `package poll
type FD struct{}
func (*FD) Write(s string) (int, error) { println(s); return len(s), nil }
`)
			add("syscall", "package syscall; var Stdin, Stdout, Stderr = 0, 1, 2")
			files := `var Stdin = NewFile(0, "/dev/stdin")
var Stdout = NewFile(1, "/dev/stdout")
var Stderr = NewFile(2, "/dev/stderr")
`
			if tc.embedded || tc.native {
				files = `var Stdin = NewFile(uintptr(syscall.Stdin), "/dev/stdin")
var Stdout = NewFile(uintptr(syscall.Stdout), "/dev/stdout")
var Stderr = NewFile(uintptr(syscall.Stderr), "/dev/stderr")
`
			}
			var fileWriter string
			if !tc.native {
				fileWriter = `func llgoWriteClosedStdout(s string) (int, error) { return new(poll.FD).Write(s) }`
			}
			add("os", `package os
import ("internal/poll"; "syscall")
type File struct{}
func NewFile(uintptr, string) *File { return new(File) }
func Dynamic() string { return "runtime format" }
func WriteFile() { new(poll.FD).Write("file") }
func llgoWriteStdoutDescriptor(s string) (int, error) { println(s); return len(s), nil }
func llgoWriteStdoutDiscard(s string) (int, error) { return len(s), nil }
`+files+fileWriter)
			add("fmt", `package fmt
import "os"
func Printf(s string, a ...any) (int, error) { return llgoPrintfLiteral(s) }
func llgoPrintfLiteral(s string) (int, error) { println(os.Stdout, s); return len(s), nil }
`)
			var rawAlias string
			if tc.rawAlias {
				rawAlias = "\n//go:linkname escaped syscall.Stdout\n"
			}
			entry := add("main", `package main
import ("fmt"; "os"; "syscall")
var escaped *int
func save(p *int) { escaped = p }
func main() {`+tc.body+"}"+rawAlias)
			if tc.embedded {
				entry.Dir = filepath.Join(goroot, "app")
			}
			prog.Build()
			goos, goarch := "wasip1", "wasm"
			if tc.native {
				goos, goarch = "darwin", "arm64"
			}
			backend := llssa.NewProgram(&llssa.Target{GOOS: goos, GOARCH: goarch, WasmReflectBridges: tc.reflectiveMethods})
			defer backend.Dispose()
			ctx := &context{
				prog: backend, progSSA: prog, initial: []*packages.Package{entry}, pkgByID: pkgs,
				wasmClosedProgram: true,
				crossCompile: crosscompile.Export{
					WasmProvider: crosscompile.WasmProviderWASI, WasmProfile: crosscompile.WasmProfileW32,
				},
				buildConf: &Config{
					Mode: ModeBuild, BuildMode: BuildModeExe, Goos: goos, Goarch: goarch, WasmOpt: WasmOptSize,
					LTO: lto.Full, OptLevel: optlevel.Oz, DeadcodeDrop: true, DisableGoGlobalDCE: true, Verbose: true,
				},
			}
			if tc.embedded {
				ctx.buildConf.Goos, ctx.buildConf.Goarch, ctx.buildConf.Target = "linux", "arm", "cortex-m-mps2"
				ctx.crossCompile = crosscompile.Export{ExtraFiles: embeddedBootFiles(ctx.buildConf.Target)}
				ctx.wasmClosedProgram, ctx.embeddedClosedProgram = false, true
			}
			if tc.native {
				ctx.crossCompile = crosscompile.Export{}
				ctx.wasmClosedProgram, ctx.nativeClosedProgram = false, true
				ctx.sizePruningRoots = tc.assembly
			}
			if err := recordPackageSSAInstructions(ctx); err != nil {
				t.Fatal(err)
			}
			plan := ctx.stdout
			if plan == nil {
				t.Fatal("missing stdout plan")
			}
			configureOptionalInitializers(ctx, goroot, &linkPruningInputs{roots: tc.assembly})
			callee := plan.regular
			if tc.writer != "" {
				callee = prog.ImportedPackage("os").Func(tc.writer)
			}
			if want := tc.writer != ""; (plan.selected != nil) != want {
				t.Fatalf("selected = %v, want specialization %v", plan.selected, want)
			}
			if tc.writer != "" {
				want := 3
				if tc.native {
					want++
				}
				if len(ctx.initializerGuardsOff) != want {
					t.Fatalf("specialization selected unexpected initializers: %v", ctx.initializerGuardsOff)
				}
			}
			for _, call := range plan.calls {
				if got := call.call.Common().StaticCallee(); got != callee {
					t.Fatalf("callee = %v, want %v", got, callee)
				}
			}
			if tc.embedded {
				if tc.externalFlags {
					ctx.commands.environ = []string{"CCFLAGS=-DEXTERNAL"}
				}
				configureEmbeddedStartup(ctx, goroot, &linkPruningInputs{roots: tc.assembly})
				if ctx.probeEmbeddedStartup != tc.compact {
					t.Fatalf("compact startup=%t, want %t", ctx.probeEmbeddedStartup, tc.compact)
				}
			}
		})
	}
}
