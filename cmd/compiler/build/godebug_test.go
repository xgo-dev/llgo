package build

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xgo-dev/llgo/types"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llgo/cmd/compiler/lto"
	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	gopackages "golang.org/x/tools/go/packages"
)

const godebugMainSource = `package main
import "math/rand"
func seeded() bool { rand.Seed(1); a := rand.Int63(); rand.Seed(1); return a == rand.Int63() }
var initialized = seeded()
func main() { println(initialized, seeded()) }
`

// Exercise the selected-toolchain metadata boundary independently of LLVM:
// older Go omits the field, newer Go streams multiple package records, and
// subprocess or metadata errors must retain useful diagnostics.
func TestDefaultGODEBUGMetadata(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	writeBuildTestTool(t, bin, "go")
	dir := filepath.Join(t.TempDir(), "directory with spaces")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	argsFile := filepath.Join(t.TempDir(), "args.json")
	for _, tc := range []struct{ name, data, want, failure string }{
		{"current", `{"ImportPath":"example.com/other","DefaultGODEBUG":"ignored=1"}` + "\n" + `{"ImportPath":"example.com/app.test","DefaultGODEBUG":"randseednop=0"}`, "randseednop=0", ""},
		{"older", `{"ImportPath":"example.com/app.test"}`, "", ""},
		{"invalid-json", "not JSON", "", "decode default GODEBUG"},
		{"package-error", `{"ImportPath":"example.com/app.test","Error":{"Err":"invalid //go:debug directive"}}`, "", "invalid //go:debug directive"},
		{"command-error", "exit", "", "Go metadata unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &packages.Config{Dir: dir, Tests: true, BuildFlags: []string{"-tags=llgo,custom", "-mod=readonly"},
				Env: withEnv(os.Environ(), "LLGO_TEST_GODEBUG_METADATA="+tc.data, "LLGO_TEST_GODEBUG_ARGS="+argsFile)}
			roots := []*packages.Package{{ID: "example.com/app.test", Name: "main"}, {ID: "example.com/other", Name: "library"}}
			defaults, err := resolveDefaultGODEBUG(cfg, root, roots, []string{"./..."})
			if tc.failure != "" {
				if err == nil || !strings.Contains(err.Error(), tc.failure) {
					t.Fatalf("metadata error: %v, want %q", err, tc.failure)
				}
				if tc.data == "exit" {
					var exit *exec.ExitError
					if !errors.As(err, &exit) || exit.ExitCode() != 7 {
						t.Fatalf("toolchain failure lost exit status: %v", err)
					}
				}
				return
			}
			if err != nil || len(defaults) != 1 || defaults[roots[0].ID] != tc.want {
				t.Fatalf("metadata defaults: %v, %v", defaults, err)
			}
			var invocation struct {
				Args []string
				PWD  string
			}
			data, err := os.ReadFile(argsFile)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &invocation); err != nil {
				t.Fatal(err)
			}
			want := []string{"list", "-e", "-json", "-test", "-tags=llgo,custom", "-mod=readonly", "--", "example.com/app"}
			if !slices.Equal(invocation.Args, want) || invocation.PWD != dir {
				t.Fatalf("selected-toolchain invocation: %+v", invocation)
			}
		})
	}
	if defaults, err := resolveDefaultGODEBUG(&packages.Config{}, "missing", []*packages.Package{{Name: "library"}}, nil); err != nil || len(defaults) != 0 {
		t.Fatalf("library unnecessarily queried entry metadata: %v, %v", defaults, err)
	}
}

func runDefaultGODEBUGMetadataHelper() {
	data := os.Getenv("LLGO_TEST_GODEBUG_METADATA")
	if data == "exit" {
		fmt.Fprintln(os.Stderr, "Go metadata unavailable")
		os.Exit(7)
	}
	record, _ := json.Marshal(struct {
		Args []string
		PWD  string
	}{os.Args[1:], os.Getenv("PWD")})
	if err := os.WriteFile(os.Getenv("LLGO_TEST_GODEBUG_ARGS"), record, 0600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(8)
	}
	fmt.Println(data)
}

func godebugFixture(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("builds GODEBUG applications")
	}
	repo, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", repo)
	t.Setenv("GOENV", "off")
	t.Setenv("GOFLAGS", "")
	t.Setenv("GOTOOLCHAIN", "local")
	t.Setenv("GOWORK", "")
	t.Setenv("GODEBUG", "")
	t.Setenv(llgoBuildCache, "1")
	return t.TempDir()
}

func godebugConfig(mode Mode) *Config {
	return &Config{Mode: mode, BuildMode: BuildModeExe, Goos: runtime.GOOS, Goarch: runtime.GOARCH, OptLevel: optlevel.O0}
}

func godebugGo(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func godebugAssertProgram(t *testing.T, app, want string) {
	t.Helper()
	output, err := exec.Command(app).CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != want {
		t.Fatalf("GODEBUG application: %v, output %q, want %q", err, output, want)
	}
}

func TestDefaultGODEBUGBuildMetadataError(t *testing.T) {
	dir := godebugFixture(t)
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/app\ngo 1.27.0\n")
	source := filepath.Join(dir, "main.go")
	writeFile(t, source, "package main\nfunc main() {}\n")
	// An external package driver can load a main that the selected Go
	// toolchain cannot resolve. Propagate that metadata error through Build.
	const id = "example.com/driver-only"
	response, err := json.Marshal(gopackages.DriverResponse{Compiler: "gc", Arch: runtime.GOARCH, GoVersion: 27,
		Roots: []string{id}, Packages: []*gopackages.Package{{ID: id, PkgPath: id, Name: "main",
			GoFiles: []string{source}, CompiledGoFiles: []string{source}}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOPACKAGESDRIVER", writeBuildTestTool(t, t.TempDir(), "driver"))
	t.Setenv("LLGO_TEST_GODEBUG_METADATA", string(response))
	t.Setenv("LLGO_TEST_GODEBUG_ARGS", filepath.Join(t.TempDir(), "args.json"))
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOSUMDB", "off")
	_, err = Build(Invocation{Dir: dir, Config: godebugConfig(ModeBuild)})
	if err == nil || !strings.Contains(err.Error(), "default GODEBUG for "+id+":") || !strings.Contains(err.Error(), "no required module provides package") {
		t.Fatalf("Build did not propagate the Go metadata diagnostic: %v", err)
	}
}

func TestDefaultGODEBUGGOPATHLocalPackages(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.go"), "//go:debug randseednop=0\npackage main\nfunc main() {}\n")
	writeFile(t, filepath.Join(dir, "main_test.go"), "//go:debug randseednop=0\npackage main\nimport \"testing\"\nfunc TestLocal(t *testing.T) {}\n")
	environ := withEnv(os.Environ(), "GOENV=off", "GOFLAGS=", "GO111MODULE=off", "GOWORK=off",
		"GOPATH="+t.TempDir(), "GOTOOLCHAIN=local", "GOPACKAGESDRIVER=off")
	for _, tests := range []bool{false, true} {
		for _, pattern := range []string{".", "./..."} {
			t.Run(fmt.Sprintf("tests=%t/%s", tests, pattern), func(t *testing.T) {
				cfg := &packages.Config{Dir: dir, Env: environ, Tests: tests, Mode: packages.NeedName}
				roots, err := packages.LoadMetadata(cfg, pattern)
				if err != nil {
					t.Fatal(err)
				}
				defaults, err := resolveDefaultGODEBUG(cfg, runtime.GOROOT(), roots, []string{pattern})
				if err != nil {
					t.Fatal(err)
				}
				entries := 0
				for _, pkg := range roots {
					if pkg.Name != "main" || (tests && !strings.HasSuffix(pkg.ID, ".test")) {
						continue
					}
					entries++
					if !strings.HasPrefix(pkg.ID, "_/") || !strings.Contains(defaults[pkg.ID], "randseednop=0") {
						t.Fatalf("local entry %q defaults = %q", pkg.ID, defaults[pkg.ID])
					}
				}
				if entries == 0 {
					t.Fatal("no GOPATH-local entry was checked")
				}
			})
		}
	}
}

func TestDefaultGODEBUGPrecedenceAndCache(t *testing.T) {
	root := godebugFixture(t)
	for _, tc := range []struct {
		name, mod, work, source, env, want string
	}{
		{"module-version", "go 1.23", "", "", "", "true true"},
		{"current-version", "go 1.27.0", "", "", "", "false false"},
		{"default-version", "go 1.27.0\ngodebug default=go1.23", "", "", "", "true true"},
		{"module-directive", "go 1.27.0\ngodebug randseednop=0", "", "", "", "true true"},
		{"workspace-version", "go 1.27.0\ngodebug randseednop=0", "go 1.27.0", "", "", "false false"},
		{"workspace-directive", "go 1.27.0\ngodebug randseednop=1", "go 1.27.0\ngodebug randseednop=0", "", "", "true true"},
		{"source-directive", "go 1.27.0", "go 1.27.0\ngodebug randseednop=1", "//go:debug randseednop=0\n", "", "true true"},
		{"source-change", "go 1.27.0", "go 1.27.0\ngodebug randseednop=0", "//go:debug randseednop=1\n", "", "false false"},
		{"environment-zero", "go 1.27.0", "go 1.27.0", "//go:debug randseednop=1\n", "randseednop=0", "true true"},
		{"environment-one", "go 1.27.0", "go 1.27.0\ngodebug randseednop=0", "", "randseednop=1", "false false"},
		{"remove-workspace-default", "go 1.27.0", "go 1.27.0", "", "", "false false"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writeFile(t, filepath.Join(root, "go.mod"), "module example.com/debugapp\n"+tc.mod+"\n")
			writeFile(t, filepath.Join(root, "main.go"), tc.source+godebugMainSource)
			if tc.work == "" {
				_ = os.Remove(filepath.Join(root, "go.work"))
			} else {
				writeFile(t, filepath.Join(root, "go.work"), tc.work+"\nuse .\n")
			}
			t.Setenv("GODEBUG", tc.env)
			if got := godebugGo(t, root, "run", "."); got != tc.want {
				t.Fatalf("Go baseline %q, want %q", got, tc.want)
			}
			conf := godebugConfig(ModeBuild)
			conf.OutFile = filepath.Join(t.TempDir(), "app"+defaultAppExt(conf))
			if _, err := Build(Invocation{Dir: root, Config: conf}); err != nil {
				t.Fatal(err)
			}
			godebugAssertProgram(t, conf.OutFile, tc.want)
		})
	}
}

func TestDefaultGODEBUGMultipleExecutables(t *testing.T) {
	root := godebugFixture(t)
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/debugmulti\ngo 1.27.0\n")
	for name, value := range map[string]string{"first": "0", "second": "1"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(root, name, "main.go"), "//go:debug randseednop="+value+"\n"+godebugMainSource)
	}
	conf := godebugConfig(ModeBuild)
	conf.OutFile = filepath.Join(root, "bin") + string(os.PathSeparator)
	if _, err := Build(Invocation{Dir: root, Args: []string{"./first", "./second"}, Config: conf}); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"first": "true true", "second": "false false"} {
		if got := godebugGo(t, root, "run", "./"+name); got != want {
			t.Fatalf("Go baseline %q, want %q", got, want)
		}
		godebugAssertProgram(t, filepath.Join(root, "bin", name+defaultAppExt(conf)), want)
	}
}

func TestDefaultGODEBUGFileArguments(t *testing.T) {
	root := godebugFixture(t)
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/debugfiles\ngo 1.27.0\n")
	writeFile(t, filepath.Join(root, "main.go"), "//go:debug randseednop=0\n"+godebugMainSource)
	if got := godebugGo(t, root, "run", "main.go"); got != "true true" {
		t.Fatalf("Go file baseline %q", got)
	}
	conf := godebugConfig(ModeBuild)
	conf.OutFile = filepath.Join(root, "app"+defaultAppExt(conf))
	if _, err := Build(Invocation{Dir: root, Args: []string{"main.go"}, Config: conf}); err != nil {
		t.Fatal(err)
	}
	godebugAssertProgram(t, conf.OutFile, "true true")

	writeFile(t, filepath.Join(root, "main_test.go"), `//go:debug randseednop=0
package main
import ("math/rand"; "testing")
func seeded() bool { rand.Seed(1); a := rand.Int63(); rand.Seed(1); return a == rand.Int63() }
var initialized = seeded()
func TestDefaults(t *testing.T) { if !initialized || !seeded() { t.Fatal("file test defaults not initialized") } }
`)
	godebugGo(t, root, "test", "-count=1", "main_test.go")
	if _, err := Build(Invocation{Dir: root, Args: []string{"main_test.go"}, Config: godebugConfig(ModeTest)}); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultGODEBUGTestMain(t *testing.T) {
	root := godebugFixture(t)
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/debugtest\ngo 1.27.0\n")
	writeFile(t, filepath.Join(root, "go.work"), "go 1.27.0\nuse .\ngodebug randseednop=1\n")
	writeFile(t, filepath.Join(root, "value.go"), "package value\n")
	const source = `//go:debug randseednop=0
package value
import ("math/rand"; "testing")
func seeded() bool { rand.Seed(1); a := rand.Int63(); rand.Seed(1); return a == rand.Int63() }
var initialized = seeded()
func TestDefaults(t *testing.T) { if !initialized || !seeded() { t.Fatal("test main defaults not initialized") } }
`
	writeFile(t, filepath.Join(root, "value_test.go"), source)
	godebugGo(t, root, "test", "-count=1", ".")
	if _, err := Build(Invocation{Dir: root, Config: godebugConfig(ModeTest)}); err != nil {
		t.Fatal(err)
	}
	// External test directives contribute to the same generated main.
	writeFile(t, filepath.Join(root, "value_test.go"), strings.TrimPrefix(source, "//go:debug randseednop=0\n"))
	writeFile(t, filepath.Join(root, "external_test.go"), strings.Replace(source, "package value", "package value_test", 1))
	if err := os.Mkdir(filepath.Join(root, "other"), 0755); err != nil {
		t.Fatal(err)
	}
	other := strings.ReplaceAll(source, "randseednop=0", "randseednop=1")
	other = strings.ReplaceAll(other, "package value", "package other")
	other = strings.ReplaceAll(other, "!initialized || !seeded()", "initialized || seeded()")
	writeFile(t, filepath.Join(root, "other", "other_test.go"), other)
	godebugGo(t, root, "test", "-count=1", "./...")
	if _, err := Build(Invocation{Dir: root, Args: []string{"./..."}, Config: godebugConfig(ModeTest)}); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultGODEBUGSourceSelection(t *testing.T) {
	root := godebugFixture(t)
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/debugselect\ngo 1.27.0\n")
	writeFile(t, filepath.Join(root, "main.go"), "//go:build !debugdefaults\n//go:debug randseednop=1\n"+godebugMainSource)
	writeFile(t, filepath.Join(root, "selected.go"), "//go:build debugdefaults\n//go:debug randseednop=0\n"+godebugMainSource)
	conf := godebugConfig(ModeBuild)
	conf.Tags = "debugdefaults"
	conf.OutFile = filepath.Join(root, "app"+defaultAppExt(conf))
	if got := godebugGo(t, root, "run", "-tags=debugdefaults", "."); got != "true true" {
		t.Fatalf("Go baseline %q", got)
	}
	if _, err := Build(Invocation{Dir: root, Config: conf}); err != nil {
		t.Fatal(err)
	}
	godebugAssertProgram(t, conf.OutFile, "true true")

	// A caller's Go overlay must contribute directives as well as Go code.
	writeFile(t, filepath.Join(root, "selected.go"), "//go:build debugdefaults\n//go:debug randseednop=1\n"+godebugMainSource)
	overlaid := filepath.Join(t.TempDir(), "selected.go")
	writeFile(t, overlaid, "//go:build debugdefaults\n//go:debug randseednop=0\n"+godebugMainSource)
	data, err := json.Marshal(struct{ Replace map[string]string }{map[string]string{filepath.Join(root, "selected.go"): overlaid}})
	if err != nil {
		t.Fatal(err)
	}
	overlay := filepath.Join(root, "overlay.json")
	writeFile(t, overlay, string(data))
	conf.GoBuildFlags = []string{"-overlay=" + overlay}
	if got := godebugGo(t, root, "run", "-tags=debugdefaults", "-overlay="+overlay, "."); got != "true true" {
		t.Fatalf("Go overlay baseline %q", got)
	}
	if _, err := Build(Invocation{Dir: root, Config: conf}); err != nil {
		t.Fatal(err)
	}
	godebugAssertProgram(t, conf.OutFile, "true true")
}

func TestDefaultGODEBUGOptimized(t *testing.T) {
	root := godebugFixture(t)
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/debugopt\ngo 1.27.0\ngodebug randseednop=0\n")
	writeFile(t, filepath.Join(root, "main.go"), godebugMainSource)
	for _, mode := range []lto.Mode{lto.Off, lto.Thin, lto.Full} {
		t.Run(mode.String(), func(t *testing.T) {
			conf := godebugConfig(ModeBuild)
			conf.OptLevel, conf.LTO = optlevel.O2, mode
			conf.OutFile = filepath.Join(t.TempDir(), "app"+defaultAppExt(conf))
			if _, err := Build(Invocation{Dir: root, Config: conf}); err != nil {
				t.Fatal(err)
			}
			godebugAssertProgram(t, conf.OutFile, "true true")
		})
	}
}

func TestDefaultGODEBUGEntryInitialization(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "windows"} {
		for _, mode := range []BuildMode{BuildModeExe, BuildModeCArchive, BuildModeCShared} {
			t.Run(goos+"/"+string(mode), func(t *testing.T) {
				ctx := &context{prog: llvmgen.NewProgram(nil), buildConf: &Config{Goos: goos, Goarch: "amd64", BuildMode: mode}}
				defer ctx.prog.Dispose()
				rt := types.NewPackage(llvmgen.PkgRuntime, "runtime")
				str := types.NewNamed(types.NewTypeName(token.NoPos, rt, "String", nil), types.NewStruct([]*types.Var{
					types.NewField(token.NoPos, rt, "Data", types.NewPointer(types.Typ[types.Byte]), false),
					types.NewField(token.NoPos, rt, "Len", types.Typ[types.Int], false),
				}, nil), nil)
				rt.Scope().Insert(str.Obj())
				ctx.prog.SetRuntime(rt)
				pkg := &packages.Package{PkgPath: "example.com/debugentry"}
				cfg := &genConfig{rtInit: true, defaultGODEBUG: "randseednop=0", abiSymbols: map[string]none{"runtime.godebugDefault": {}}}
				entry := genMainModule(ctx, llvmgen.PkgRuntime, pkg, cfg)
				ir := entry.LPkg.String()
				assertInOrder(t, ir, "call void @__llgo_godebug_init()", `call void @"github.com/xgo-dev/llgo/runtime/internal/runtime.init"()`)
				if !strings.Contains(ir, "@runtime.godebugDefault = external global") {
					t.Fatalf("default setter must reference shared runtime storage:\n%s", ir)
				}
			})
		}
	}
}

func TestDefaultGODEBUGExplicitGOWORKInvalidation(t *testing.T) {
	root := godebugFixture(t)
	extDir := t.TempDir()
	extWork := filepath.Join(extDir, "go.work")
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/debugextwork\ngo 1.27.0\n")
	writeFile(t, filepath.Join(root, "main.go"), godebugMainSource)
	writeFile(t, extWork, "go 1.27.0\nuse "+root+"\ngodebug randseednop=0\n")

	t.Setenv("GOWORK", extWork)
	conf := godebugConfig(ModeBuild)
	conf.OutFile = filepath.Join(t.TempDir(), "app"+defaultAppExt(conf))
	if _, err := Build(Invocation{Dir: root, Config: conf}); err != nil {
		t.Fatal(err)
	}
	godebugAssertProgram(t, conf.OutFile, "true true")

	// Now modify the external go.work file to randseednop=1
	writeFile(t, extWork, "go 1.27.0\nuse "+root+"\ngodebug randseednop=1\n")
	conf2 := godebugConfig(ModeBuild)
	conf2.OutFile = filepath.Join(t.TempDir(), "app2"+defaultAppExt(conf2))
	if _, err := Build(Invocation{Dir: root, Config: conf2}); err != nil {
		t.Fatal(err)
	}
	godebugAssertProgram(t, conf2.OutFile, "false false")
}
