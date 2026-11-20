package build

import (
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xgo-dev/llgo/internal/buildenv"
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
