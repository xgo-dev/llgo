package build

import (
	"go/token"
	"go/types"
	"slices"
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestConfigureDarwinExecutableCwd(t *testing.T) {
	const source = `package os

var getwdCache string
var initCwd, initCwdErr = Getwd()

func Getwd() (string, error) {
	getwdCache = "cached"
	return getwdCache, nil
}
func executable() (string, error) { return initCwd, initCwdErr }
func Executable() (string, error) { return executable() }
func ReadCache() string { return getwdCache }
func ReadCwd() string { return initCwd }
`
	updated := rewriteDarwinExecutableCwd([]byte(source))
	if updated == nil || rewriteDarwinExecutableCwd(updated) != nil {
		t.Fatal("cannot apply the startup rewrite exactly once")
	}
	for _, tt := range []struct {
		name, body, root string
		want             bool
	}{
		{name: "unused", body: `println("hello")`, want: true},
		{name: "getwd", body: `os.Getwd()`},
		{name: "indirect getwd", body: `run(os.Getwd)`},
		{name: "executable", body: `os.Executable()`},
		{name: "cache read", body: `println(os.ReadCache())`},
		{name: "startup directory read", body: `println(os.ReadCwd())`},
		{name: "external variable use", root: "os.initCwdErr"},
		{name: "external function use", root: "os.Getwd"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
			dep := checkFormatTestPackage(t, prog.Fset, "os", string(updated), types.Config{})
			main := checkFormatTestPackage(t, prog.Fset, "main", `package main
import "os"
func run(fn func() (string, error)) { fn() }
func main() { `+tt.body+` }
`, types.Config{Importer: codecInitTestImporter{"os": dep.Types}, DisableUnusedImportCheck: true})
			osPkg := prog.CreatePackage(dep.Types, dep.Syntax, dep.TypesInfo, true)
			mainPkg := prog.CreatePackage(main.Types, main.Syntax, main.TypesInfo, true)
			prog.Build()

			ctx := &context{buildConf: &Config{}}
			init := osPkg.Func(darwinExecutableCwdInitializer)
			blockCount := len(init.Blocks)
			configureDarwinExecutableCwd(ctx, osPkg, []*ssa.Function{mainPkg.Func("init"), mainPkg.Func("main")},
				func(name string) bool { return name == tt.root })
			if len(init.Blocks) != blockCount {
				t.Fatal("startup body changed in the shared package")
			}
			if got := slices.Contains(ctx.initializerGuardsOff, darwinExecutableCwdGuard); got != tt.want {
				t.Fatalf("omit initial working directory = %v, want %v", got, tt.want)
			}
		})
	}
}
