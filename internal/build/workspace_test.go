package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	gopackages "golang.org/x/tools/go/packages"
)

func TestWorkspaceRuntimeConfig(t *testing.T) {
	caller := &packages.Config{Dir: "app", Env: []string{"GOWORK=/app/go.work", "GOFLAGS=-mod=vendor"}, BuildFlags: []string{"-tags=llgo", "-modfile=/app/custom.mod"}}
	if err := applyPackageLoadFlags(caller, "-mod=vendor '-tags=first second' -overlay=/app/overlay.json -pkgdir=/app/exports"); err != nil {
		t.Fatal(err)
	}
	conf := runtimePackageConfig(caller, "compiler/runtime")
	if conf.Dir != "compiler/runtime" || (commandEnv{environ: conf.Env}).lookup("GOWORK") != "off" {
		t.Fatalf("runtime configuration: %+v", conf)
	}
	want := "-mod=vendor|-tags=first second|-overlay=/app/overlay.json|-pkgdir=/app/exports|-tags=llgo|-modfile=/app/custom.mod|-mod=readonly|-modfile="
	if got := strings.Join(conf.BuildFlags, "|"); got != want {
		t.Fatalf("runtime flags %q, want %q", got, want)
	}
	if caller.Dir != "app" || (commandEnv{environ: caller.Env}).lookup("GOWORK") != "/app/go.work" || len(caller.BuildFlags) != 6 {
		t.Fatal("runtime configuration changed application inputs")
	}
}

func TestWorkspaceInvalidGoFlags(t *testing.T) {
	conf := &packages.Config{Env: []string{"GOFLAGS=keep"}, BuildFlags: []string{"-tags=llgo"}}
	if err := applyPackageLoadFlags(conf, "'-modfile=unterminated"); err == nil {
		t.Fatal("invalid quoted flags accepted")
	}
	if strings.Join(conf.Env, "|") != "GOFLAGS=keep" || strings.Join(conf.BuildFlags, "|") != "-tags=llgo" {
		t.Fatal("invalid flags changed the package configuration")
	}
}

func TestWorkspaceBuildRejectsInvalidGoFlagsMetadata(t *testing.T) {
	bin := t.TempDir()
	writeBuildTestTool(t, bin, "go")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("LLGO_TEST_GO_CONFIG_HELPER", "invalid-goflags")
	t.Setenv("GOENV", "off")
	t.Setenv("GOFLAGS", "")
	t.Setenv("GOTOOLCHAIN", "local")
	t.Setenv("GOWORK", "off")
	// The launcher supplies malformed metadata while the selected compiler
	// still resolves tool tags. Build must reject it before loading packages.
	_, err := Build(Invocation{Dir: t.TempDir(), Config: workspaceConfig(ModeBuild)})
	if err == nil || !strings.Contains(err.Error(), "parse Go package flags:") {
		t.Fatalf("invalid GOFLAGS metadata was not rejected by Build: %v", err)
	}
}

func TestWorkspacePackageListErrors(t *testing.T) {
	deferred := &packages.Package{Errors: []packages.Error{
		{Kind: gopackages.ListError, Msg: "# example.com/llgo-only\ngc cannot compile this source"},
		{Kind: gopackages.TypeError, Msg: "LLGo must type-check patched source"},
	}}
	if err := initialPackageListErrors([]*packages.Package{deferred}); err != nil {
		t.Fatalf("frontend diagnostics were not deferred: %v", err)
	}
	roots := []*packages.Package{deferred, {Errors: []packages.Error{
		{Kind: gopackages.ListError, Msg: "directory prefix does not contain workspace modules"},
		{Kind: gopackages.ListError, Msg: "missing package directory"},
	}}}
	err := initialPackageListErrors(roots)
	if err == nil || !strings.Contains(err.Error(), "workspace modules") || !strings.Contains(err.Error(), "missing package directory") || strings.Contains(err.Error(), "gc cannot") {
		t.Fatalf("package errors were lost or mixed with deferred diagnostics: %v", err)
	}
}

func workspaceFixture(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("builds workspace executables")
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", repo)
	t.Setenv("GOENV", "off")
	t.Setenv("GOFLAGS", "")
	t.Setenv("GOTOOLCHAIN", "local")
	t.Setenv("GOWORK", "")
	t.Setenv("GO111MODULE", "on")
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOSUMDB", "off")
	t.Setenv(llgoBuildCache, "1")
	root := filepath.Join(t.TempDir(), "workspace with spaces")
	files := map[string]string{
		"go.work":                    "go 1.27.0\nuse (\n ./app\n ./lib\n)\n",
		"app/go.mod":                 "module example.com/workapp\ngo 1.27.0\nrequire example.com/worklib v0.0.0\n",
		"app/cmd/hello/main.go":      "package main\nimport \"example.com/worklib\"\nfunc main(){println(lib.Value())}\n",
		"app/cmd/hello/main_test.go": "package main\nimport \"testing\"\nfunc TestMainValue(t *testing.T){main()}\n",
		"lib/go.mod":                 "module example.com/worklib\ngo 1.20\n",
		"lib/lib.go":                 "package lib\nfunc Value() int{return 41}\n",
		"lib/lib_test.go":            "package lib\nimport \"testing\"\nfunc TestValue(t *testing.T){if Value()!=41{t.Fatal(Value())}}\n",
	}
	for name, source := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, path, source)
	}
	return root
}

func workspaceConfig(mode Mode) *Config {
	return &Config{Mode: mode, Goos: runtime.GOOS, Goarch: runtime.GOARCH, OptLevel: optlevel.O0}
}

func workspaceGo(t *testing.T, dir string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go %v: %v\n%s", args, err, output)
	}
	return output
}

func workspaceBuildRun(t *testing.T, dir string, flags []string, want string) {
	t.Helper()
	conf := workspaceConfig(ModeBuild)
	conf.GoBuildFlags = flags
	conf.OutFile = filepath.Join(t.TempDir(), "app")
	if runtime.GOOS == "windows" {
		conf.OutFile += ".exe"
	}
	if _, err := Build(Invocation{Dir: dir, Config: conf}); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(conf.OutFile).CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != want {
		t.Fatalf("workspace program: %v, output %q, want %q", err, output, want)
	}
}

func TestWorkspaceBuildSelectionAndCache(t *testing.T) {
	root := workspaceFixture(t)
	dir := filepath.Join(root, "app", "cmd", "hello")
	workspaceBuildRun(t, dir, nil, "41") // Discovery from a nested member directory.
	t.Setenv("GOWORK", filepath.Join(root, "go.work"))
	workspaceBuildRun(t, dir, nil, "41")
	writeFile(t, filepath.Join(root, "lib", "lib.go"), "package lib\nfunc Value() int{return 42}\n")
	workspaceBuildRun(t, dir, nil, "42")
	if err := os.Mkdir(filepath.Join(root, "alternate"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "alternate", "go.mod"), "module example.com/worklib\ngo 1.20\n")
	writeFile(t, filepath.Join(root, "alternate", "lib.go"), "package lib\nfunc Value() int{return 75}\n")
	writeFile(t, filepath.Join(root, "selected.work"), "go 1.27.0\nuse (\n ./app\n ./alternate\n)\n")
	t.Setenv("GOWORK", filepath.Join(root, "selected.work"))
	workspaceBuildRun(t, dir, nil, "75")
	t.Setenv("GOWORK", filepath.Join(root, "go.work"))
	workspaceBuildRun(t, dir, nil, "42")
	t.Setenv("GOWORK", "off")
	if _, err := Build(Invocation{Dir: dir, Config: workspaceConfig(ModeBuild)}); err == nil {
		t.Fatal("workspace-only dependency resolved with GOWORK=off")
	}
	// A module-specific flags file must select application dependencies without
	// redirecting the compiler runtime load, including flags persisted in GOENV.
	modfile := filepath.Join(root, "app", "selected.mod")
	writeFile(t, modfile, "module example.com/workapp\ngo 1.20\nrequire example.com/worklib v0.0.0\nreplace example.com/worklib => ../lib\n")
	workspaceBuildRun(t, dir, []string{"-modfile=" + modfile}, "42")
	goenv := filepath.Join(root, "goenv")
	writeFile(t, goenv, "GOFLAGS='-modfile="+modfile+"' -tags=workspace_test\n")
	t.Setenv("GOENV", goenv)
	workspaceGo(t, dir, "list", ".")
	workspaceBuildRun(t, dir, nil, "42")
}

func TestWorkspacePackageErrors(t *testing.T) {
	root := workspaceFixture(t)
	for _, args := range [][]string{{"./..."}, {"./missing"}, {"./lib", "./missing"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if _, err := Build(Invocation{Dir: root, Args: args, Config: workspaceConfig(ModeTest)}); err == nil {
				t.Fatalf("test %v discarded package-list errors", args)
			}
		})
	}
	if err := os.Mkdir(filepath.Join(root, "lib", "empty"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(Invocation{Dir: root, Args: []string{"./lib/empty/..."}, Config: workspaceConfig(ModeTest)}); err != nil {
		t.Fatalf("valid pattern matching no packages: %v", err)
	}
	writeFile(t, filepath.Join(root, "go.work"), "go 1.999\nuse ./app\n")
	_, err := Build(Invocation{Dir: root, Args: []string{"./app/..."}, Config: workspaceConfig(ModeBuild)})
	if err == nil || !strings.Contains(err.Error(), "1.999") {
		t.Fatalf("workspace toolchain requirement was ignored: %v", err)
	}
}

func TestWorkspaceRunAndInstall(t *testing.T) {
	root := workspaceFixture(t)
	t.Setenv("GOWORK", filepath.Join(root, "go.work"))
	dir := filepath.Join(root, "app", "cmd", "hello")
	if _, err := Build(Invocation{Dir: dir, Config: workspaceConfig(ModeRun)}); err != nil {
		t.Fatal(err)
	}
	conf := workspaceConfig(ModeInstall)
	conf.BinPath = t.TempDir()
	if _, err := Build(Invocation{Dir: root, Args: []string{"./app/cmd/hello"}, Config: conf}); err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(conf.BinPath, "hello")
	if runtime.GOOS == "windows" {
		app += ".exe"
	}
	output, err := exec.Command(app).CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "41" {
		t.Fatalf("installed workspace application: %v\n%s", err, output)
	}
}

func TestWorkspaceSelectedGoDriver(t *testing.T) {
	root := workspaceFixture(t)
	bin := t.TempDir()
	writeBuildTestTool(t, bin, "go")
	parentPath := bin + string(os.PathListSeparator) + os.Getenv("PATH")
	t.Setenv("PATH", parentPath)
	t.Setenv("LLGO_TEST_GO_CONFIG_HELPER", "package-driver")
	t.Setenv("GOWORK", filepath.Join(root, "go.work"))
	dir := filepath.Join(root, "app", "cmd", "hello")
	workspaceBuildRun(t, dir, nil, "41")
	conf := workspaceConfig(ModeTest)
	conf.Coverage = &CoverageConfig{Profile: filepath.Join(t.TempDir(), "coverage.out")}
	if _, err := Build(Invocation{Dir: root, Args: []string{"./lib"}, Config: conf}); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("PATH") != parentPath {
		t.Fatal("build changed the parent PATH")
	}
}

func TestWorkspaceToolchainSwitch(t *testing.T) {
	oldRoot := olderGoTestRoot(t)
	for _, direction := range []string{"up", "down"} {
		t.Run(direction, func(t *testing.T) {
			root := workspaceFixture(t)
			bin := t.TempDir()
			name := "go"
			if runtime.GOOS == "windows" {
				name += ".exe"
			}
			for version, goRoot := range map[string]string{runtime.Version(): runtime.GOROOT(), "go1.21.13": oldRoot} {
				writeBuildTestTool(t, bin, version)
				tool := filepath.Join(bin, version)
				if runtime.GOOS == "windows" {
					tool += ".exe"
				}
				writeFile(t, tool+".root", goRoot)
			}
			launcherRoot, selectedVersion := oldRoot, runtime.Version()
			want := "41 true"
			if direction == "down" {
				launcherRoot, selectedVersion = runtime.GOROOT(), "go1.21.13"
				writeFile(t, filepath.Join(root, "go.work"), "go 1.21\ntoolchain go1.21.13\nuse (\n ./app\n ./lib\n)\n")
				writeFile(t, filepath.Join(root, "app", "go.mod"), "module example.com/workapp\ngo 1.21\nrequire example.com/worklib v0.0.0\n")
				want = "41 false"
			}
			dir := filepath.Join(root, "app", "cmd", "hello")
			writeFile(t, filepath.Join(dir, "main.go"), "package main\nimport \"example.com/worklib\"\nfunc main(){println(lib.Value(), selected)}\n")
			writeFile(t, filepath.Join(dir, "new.go"), "//go:build go1.27\n\npackage main\nconst selected = true\n")
			writeFile(t, filepath.Join(dir, "old.go"), "//go:build !go1.27\n\npackage main\nconst selected = false\n")
			t.Setenv("GOROOT", "")
			t.Setenv("GOEXPERIMENT", "")
			t.Setenv("LLGO_TEST_GO_CONFIG_HELPER", "toolchain-driver")
			t.Setenv("PATH", filepath.Join(launcherRoot, "bin")+string(os.PathListSeparator)+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("GOWORK", filepath.Join(root, "go.work"))
			if direction == "up" {
				t.Setenv("GOTOOLCHAIN", "auto")
			} else {
				t.Setenv("GOTOOLCHAIN", selectedVersion)
			}
			if got := strings.TrimSpace(string(workspaceGo(t, dir, "env", "GOVERSION"))); got != selectedVersion {
				t.Fatalf("selected Go = %s, want %s", got, selectedVersion)
			}
			if got := strings.TrimSpace(string(workspaceGo(t, dir, "run", "."))); got != want {
				t.Fatalf("Go control = %q, want %q", got, want)
			}
			workspaceBuildRun(t, dir, nil, want)
		})
	}
}

func olderGoTestRoot(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	roots := []string{filepath.Join(home, "sdk", "go1.21.13")}
	if cache := os.Getenv("RUNNER_TOOL_CACHE"); cache != "" {
		arch := map[string]string{"amd64": "x64", "arm64": "arm64", "386": "x86"}[runtime.GOARCH]
		roots = append(roots, filepath.Join(cache, "go", "1.21.13", arch))
	}
	name := "go"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	for _, root := range roots {
		if _, err := os.Stat(filepath.Join(root, "bin", name)); err == nil {
			return root
		}
	}
	if os.Getenv("CI") == "true" {
		t.Fatal("Go 1.21.13 fixture is missing; CI must install it before the current Go toolchain")
	}
	t.Skip("Go 1.21.13 is not installed in the SDK or runner tool cache")
	return ""
}

func TestWorkspaceVendorAndReplace(t *testing.T) {
	root := workspaceFixture(t)
	dir := filepath.Join(root, "app", "cmd", "hello")
	// Workspace members can import each other without a published requirement.
	writeFile(t, filepath.Join(root, "app", "go.mod"), "module example.com/workapp\ngo 1.27.0\n")
	external := filepath.Join(root, "external")
	if err := os.Mkdir(external, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(external, "go.mod"), "module example.com/workext\ngo 1.20\n")
	writeFile(t, filepath.Join(external, "ext.go"), "package ext\nfunc Value() int{return 40}\n")
	writeFile(t, filepath.Join(root, "lib", "go.mod"), "module example.com/worklib\ngo 1.27.0\nrequire example.com/workext v0.0.0\nreplace example.com/workext => ../missing\n")
	writeFile(t, filepath.Join(root, "lib", "lib.go"), "package lib\nimport \"example.com/workext\"\nfunc Value() int{return ext.Value()+1}\n")
	writeFile(t, filepath.Join(root, "go.work"), "go 1.27.0\nuse (\n ./app\n ./lib\n)\nreplace example.com/workext => ./external\n")
	t.Setenv("GOWORK", filepath.Join(root, "go.work"))
	workspaceGo(t, dir, "list", "-deps", ".")
	workspaceBuildRun(t, dir, nil, "41") // Workspace replacement overrides go.mod.
	workspaceGo(t, root, "work", "vendor")
	if _, err := os.Stat(filepath.Join(root, "vendor", "example.com", "workext", "ext.go")); err != nil {
		t.Fatal(err)
	}
	// Changing the original dependency proves the vendor copy is actually used.
	writeFile(t, filepath.Join(external, "ext.go"), "package ext\nfunc Value() int{return 90}\n")
	workspaceBuildRun(t, dir, nil, "41")
	workspaceBuildRun(t, dir, []string{"-mod=vendor"}, "41")
	t.Setenv("GOFLAGS", "-mod=vendor")
	workspaceBuildRun(t, dir, nil, "41")
	t.Setenv("GOFLAGS", "")
	workspaceBuildRun(t, dir, []string{"-mod=readonly"}, "91")
	if _, err := Build(Invocation{Dir: dir, Config: &Config{Mode: ModeBuild, GoBuildFlags: []string{"-mod=mod"}}}); err == nil || !strings.Contains(err.Error(), "workspace") {
		t.Fatalf("workspace accepted -mod=mod: %v", err)
	}
}

func TestWorkspaceCoverageAgainstGo(t *testing.T) {
	root := workspaceFixture(t)
	t.Setenv("GOWORK", filepath.Join(root, "go.work"))
	goProfile, llgoProfile := filepath.Join(root, "go.out"), filepath.Join(root, "llgo.out")
	const selection = "./app/...,./lib/..."
	workspaceGo(t, root, "test", "-count=1", "-covermode=count", "-coverpkg="+selection, "-coverprofile="+goProfile, "./app/...", "./lib/...")
	conf := workspaceConfig(ModeTest)
	conf.Coverage = &CoverageConfig{Mode: "count", Packages: selection, Profile: llgoProfile}
	if _, err := Build(Invocation{Dir: root, Args: []string{"./app/...", "./lib/..."}, Config: conf}); err != nil {
		t.Fatal(err)
	}
	compareCoverageProfiles(t, goProfile, llgoProfile)
	for _, compiler := range []string{"go", "llgo"} {
		app := filepath.Join(root, compiler+"-app")
		if runtime.GOOS == "windows" {
			app += ".exe"
		}
		if compiler == "go" {
			workspaceGo(t, root, "build", "-covermode=count", "-o", app, "./app/cmd/hello")
		} else {
			conf := workspaceConfig(ModeBuild)
			conf.OutFile = app
			conf.Coverage = &CoverageConfig{Mode: "count"}
			if _, err := Build(Invocation{Dir: root, Args: []string{"./app/cmd/hello"}, Config: conf}); err != nil {
				t.Fatal(err)
			}
		}
		data := filepath.Join(root, compiler+"-data")
		if err := os.Mkdir(data, 0755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(app)
		cmd.Env = withEnv(os.Environ(), "GOCOVERDIR="+data)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s covered application: %v\n%s", compiler, err, output)
		}
		workspaceGo(t, root, "tool", "covdata", "textfmt", "-i="+data, "-o="+filepath.Join(root, compiler+"-build.out"))
	}
	compareCoverageProfiles(t, filepath.Join(root, "go-build.out"), filepath.Join(root, "llgo-build.out"))
}
