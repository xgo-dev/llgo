package packages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	gopackages "golang.org/x/tools/go/packages"
)

func TestMain(m *testing.M) {
	name := strings.TrimSuffix(strings.ToLower(filepath.Base(os.Args[0])), ".exe")
	switch name {
	case "go":
		fmt.Fprintln(os.Stderr, "unexpected PATH launcher: selected Go executable was bypassed")
		os.Exit(19)
	case "gopackagesdriver", "custom-driver":
		if len(os.Args) > 1 && os.Args[1] == "driver:wait" {
			time.Sleep(30 * time.Second)
		}
		response := gopackages.DriverResponse{NotHandled: true}
		if len(os.Args) > 1 && (os.Args[1] == "driver:handled" || os.Args[1] == "driver:gccgo") {
			response = gopackages.DriverResponse{
				Compiler: "gc", Arch: "386",
				Roots: []string{"external"}, Packages: []*Package{{ID: "external", Name: "external", PkgPath: "example.com/external"}},
			}
			if os.Args[1] == "driver:gccgo" {
				response.Compiler, response.Arch = "gccgo", "arm"
			}
		}
		if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func copyDriverTestTool(t *testing.T, dir, name string) string {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func metadataTestConfig(t *testing.T) *Config {
	t.Helper()
	dir := t.TempDir()
	writeLoadTestFile(t, filepath.Join(dir, "go.mod"), "module example.com/driver\ngo 1.20\n")
	writeLoadTestFile(t, filepath.Join(dir, "on.go"), "//go:build goexperiment.greenteagc\n\npackage driver\nconst Enabled = true\n")
	writeLoadTestFile(t, filepath.Join(dir, "off.go"), "//go:build !goexperiment.greenteagc\n\npackage driver\nconst Enabled = false\n")
	environ := os.Environ()
	for key, value := range map[string]string{
		"GOROOT": runtime.GOROOT(), "GOENV": "off", "GOWORK": "off", "GOFLAGS": "",
		"GOTOOLCHAIN": "local", "GO111MODULE": "on", "GOPACKAGESDRIVER": "off",
		"GOPROXY": "off", "GOSUMDB": "off", "GOEXPERIMENT": "none",
	} {
		environ = replaceEnvironment(environ, key, value)
	}
	return &Config{
		Dir: dir, Env: environ,
		Mode: NeedName | NeedFiles | NeedCompiledGoFiles | NeedImports | NeedDeps | NeedModule | NeedTypesSizes,
	}
}

func TestLoadMetadataConcurrentConfigurations(t *testing.T) {
	bin := t.TempDir()
	copyDriverTestTool(t, bin, "go")
	parentPath := bin + string(os.PathListSeparator) + os.Getenv("PATH")
	t.Setenv("PATH", parentPath)
	base := metadataTestConfig(t)
	original := append([]string(nil), base.Env...)
	var wg sync.WaitGroup
	for _, experiment := range []string{"none", "none,greenteagc"} {
		for _, arch := range []string{runtime.GOARCH, "386"} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cfg := *base
				cfg.Env = replaceEnvironment(cfg.Env, "GOEXPERIMENT", experiment)
				cfg.Env = replaceEnvironment(cfg.Env, "GOARCH", arch)
				cfg.Env = replaceEnvironment(cfg.Env, "GOOS", "linux")
				cfg.Env = replaceEnvironment(cfg.Env, "CGO_ENABLED", "0")
				pkgs, err := LoadMetadata(&cfg, ".")
				if err != nil {
					t.Errorf("%s/%s: %v", experiment, arch, err)
					return
				}
				if len(pkgs) != 1 || len(pkgs[0].Errors) != 0 {
					t.Errorf("%s/%s: packages = %v", experiment, arch, pkgs)
					return
				}
				wantFile := "off.go"
				if experiment != "none" {
					wantFile = "on.go"
				}
				pkg := pkgs[0]
				if len(pkg.CompiledGoFiles) != 1 || filepath.Base(pkg.CompiledGoFiles[0]) != wantFile {
					t.Errorf("%s/%s: files = %v", experiment, arch, pkg.CompiledGoFiles)
				}
				if pkg.Module == nil || pkg.Module.Path != "example.com/driver" || pkg.Module.GoVersion != "1.20" {
					t.Errorf("module metadata lost: %+v", pkg.Module)
				}
				if pkg.TypesSizes == nil || pkg.TypesSizes.Sizeof(types.Typ[types.Uintptr]) != types.SizesFor("gc", arch).Sizeof(types.Typ[types.Uintptr]) {
					t.Errorf("%s: target sizes lost", arch)
				}
			}()
		}
	}
	wg.Wait()
	if os.Getenv("PATH") != parentPath || !reflect.DeepEqual(base.Env, original) {
		t.Fatal("package loads changed their caller's environment")
	}
}

func TestLoadMetadataWorkspaceOverlayTests(t *testing.T) {
	cfg := metadataTestConfig(t)
	root := cfg.Dir
	for name, contents := range map[string]string{
		"go.work":              "go 1.27.0\nuse (\n ./app\n ./lib\n)\n",
		"app/go.mod":           "module example.com/app\ngo 1.24\n",
		"lib/go.mod":           "module example.com/lib\ngo 1.20\n",
		"lib/lib.go":           "package lib\nconst Value = 42\n",
		"app/app.go":           "package app\nimport _ \"example.com/missing\"\n",
		"app/data.txt":         "embedded data",
		"app/tagged.go":        "//go:build driver_test\n\npackage app\nconst Tagged = true\n",
		"app/app_test.go":      "package app\nimport \"testing\"\nfunc TestValue(t *testing.T) { if Value != 42 { t.Fatal(Value) } }\n",
		"app/external_test.go": "package app_test\nimport (\"testing\"; \"example.com/app\")\nfunc TestExternal(t *testing.T) { if app.Value != 42 { t.Fatal(app.Value) } }\n",
	} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		writeLoadTestFile(t, path, contents)
	}
	cfg.Dir = filepath.Join(root, "app")
	cfg.Env = replaceEnvironment(cfg.Env, "GOWORK", filepath.Join(root, "go.work"))
	cfg.Tests = true
	cfg.Mode |= NeedForTest | NeedEmbedFiles | NeedEmbedPatterns | NeedExportFile
	cfg.BuildFlags = []string{"-tags=driver_test"}
	cfg.Overlay = map[string][]byte{
		filepath.Join(cfg.Dir, "app.go"): []byte("package app\nimport (\"example.com/lib\"; _ \"embed\")\nconst Value = lib.Value\n//go:embed data.txt\nvar Data string\n"),
	}
	var logs []string
	cfg.Logf = func(format string, args ...any) { logs = append(logs, fmt.Sprintf(format, args...)) }
	pkgs, err := LoadMetadata(cfg, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 4 || len(logs) == 0 {
		t.Fatalf("roots = %v, driver logs = %v", pkgs, logs)
	}
	var main, ordinary, augmented, external *Package
	for _, pkg := range pkgs {
		if len(pkg.Errors) != 0 {
			t.Fatalf("%s: %v", pkg.ID, pkg.Errors)
		}
		switch {
		case pkg.Name == "main":
			main = pkg
		case pkg.Name == "app_test":
			external = pkg
		case pkg.ForTest != "":
			augmented = pkg
		default:
			ordinary = pkg
		}
	}
	if main == nil || ordinary == nil || augmented == nil || external == nil {
		t.Fatalf("test package identities lost: %v", pkgs)
	}
	if augmented.ForTest != "example.com/app" || external.ForTest != "example.com/app" || main.Imports["example.com/app"] != augmented {
		t.Fatal("test-main imports or ForTest metadata lost")
	}
	if ordinary.Imports["example.com/lib"] != augmented.Imports["example.com/lib"] || ordinary.Imports["example.com/lib"].Module.GoVersion != "1.20" {
		t.Fatal("workspace dependency identity or module language version lost")
	}
	if ordinary.Module == nil || ordinary.Module.GoVersion != "1.24" || ordinary.Dir != cfg.Dir || ordinary.ExportFile == "" {
		t.Fatalf("application metadata lost: %+v", ordinary)
	}
	if len(ordinary.EmbedFiles) != 1 || filepath.Base(ordinary.EmbedFiles[0]) != "data.txt" || len(ordinary.EmbedPatterns) != 1 {
		t.Fatalf("embed metadata lost: %v / %v", ordinary.EmbedFiles, ordinary.EmbedPatterns)
	}
	if len(ordinary.CompiledGoFiles) != 2 {
		t.Fatalf("build flags or overlay lost: %v", ordinary.CompiledGoFiles)
	}
	// The same metadata must survive LLGo's own overlay-aware typecheck pass.
	cfg.Tests = false
	cfg.Mode = NeedName | NeedDeps | NeedTypes | NeedTypesInfo | NeedTypesSizes | NeedSyntax | NeedModule
	pkgs, err = LoadEx(NewDeduper(), nil, cfg, ".")
	if err != nil || len(pkgs) != 1 || pkgs[0].IllTyped || len(pkgs[0].Errors) != 0 {
		t.Fatalf("overlay typecheck failed: %v / %v", err, pkgs)
	}
	if got := pkgs[0].Types.Scope().Lookup("Value").(*types.Const).Val().String(); got != "42" {
		t.Fatalf("overlay Value = %s", got)
	}
}

func TestLoadMetadataExternalDriver(t *testing.T) {
	bin := t.TempDir()
	copyDriverTestTool(t, bin, "go")
	driver := copyDriverTestTool(t, bin, "gopackagesdriver")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	cfg := metadataTestConfig(t)
	cfg.Mode = NeedName | NeedFiles | NeedImports | NeedTypesSizes
	relDir := "relative driver with spaces"
	if err := os.Mkdir(filepath.Join(cfg.Dir, relDir), 0755); err != nil {
		t.Fatal(err)
	}
	relative := copyDriverTestTool(t, filepath.Join(cfg.Dir, relDir), "custom-driver")
	relative, err := filepath.Rel(cfg.Dir, relative)
	if err != nil {
		t.Fatal(err)
	}
	for _, selection := range []string{driver, "gopackagesdriver", "", relative} {
		cfg.Env = replaceEnvironment(cfg.Env, "GOPACKAGESDRIVER", selection)
		// x/tools sets cmd.Dir = cfg.Dir even for relative driver paths.
		// Compare its original behavior with the worker, whose cwd is cfg.Dir.
		baseline, err := gopackages.Load(cfg, "driver:handled")
		if err != nil || len(baseline) != 1 || baseline[0].ID != "external" {
			t.Fatalf("original external driver %q: %v / %v", selection, err, baseline)
		}
		pkgs, err := LoadMetadata(cfg, "driver:handled")
		if err != nil || len(pkgs) != 1 || pkgs[0].ID != "external" {
			t.Fatalf("external driver %q: %v / %v", selection, err, pkgs)
		}
		if pkgs[0].TypesSizes != types.SizesFor("gc", "386") {
			t.Fatal("external driver's target sizes changed to the Go toolchain's host sizes")
		}
		pkgs, err = LoadMetadata(cfg, "driver:gccgo")
		if err != nil || len(pkgs) != 1 || !reflect.DeepEqual(pkgs[0].TypesSizes, types.SizesFor("gccgo", "arm")) {
			t.Fatalf("external gccgo driver sizes: %v / %v", err, pkgs)
		}
		pkgs, err = LoadMetadata(cfg, ".")
		if err != nil || len(pkgs) != 1 || len(pkgs[0].Errors) != 0 {
			t.Fatalf("NotHandled fallback %q: %v / %v", selection, err, pkgs)
		}
	}
}

func TestLoadMetadataErrorsAndCancellation(t *testing.T) {
	cfg := metadataTestConfig(t)
	if _, err := LoadMetadata(cfg, "unknown=query"); err == nil || !strings.Contains(err.Error(), "invalid query type") {
		t.Fatalf("driver error was lost: %v", err)
	}
	pkgs, err := LoadMetadata(cfg, "./missing")
	if err != nil || len(pkgs) != 1 || len(pkgs[0].Errors) == 0 {
		t.Fatalf("package error was lost: %v / %v", err, pkgs)
	}
	invalid := *cfg
	invalid.Mode |= NeedTypes
	if _, err := LoadMetadata(&invalid, "."); err == nil {
		t.Fatal("metadata driver accepted a typecheck request")
	}
	invalid = *cfg
	invalid.Env = replaceEnvironment(invalid.Env, "GOROOT", t.TempDir())
	if _, err := LoadMetadata(&invalid, "."); err == nil || !strings.Contains(err.Error(), "selected Go executable") {
		t.Fatalf("missing selected Go fell back to PATH: %v", err)
	}
	invalid = *cfg
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	invalid.Context = ctx
	if _, err := LoadMetadata(&invalid, "."); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled request: %v", err)
	}
	invalid = *cfg
	invalid.Env = replaceEnvironment(invalid.Env, "GOPACKAGESDRIVER", copyDriverTestTool(t, t.TempDir(), "custom-driver"))
	ctx, cancel = context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	invalid.Context = ctx
	start := time.Now()
	if _, err := LoadMetadata(&invalid, "driver:wait"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("running driver cancellation: %v", err)
	}
	if time.Since(start) > 4*time.Second {
		t.Fatal("driver subprocess did not finish cancellation promptly")
	}
}

func TestPackageMetadataRoundTrip(t *testing.T) {
	dep := &Package{ID: "dependency", PkgPath: "example.com/dep", Name: "dep"}
	root := &Package{ID: "root", PkgPath: "example.com/root", Name: "root", Imports: map[string]*Package{"example.com/dep": dep}}
	root.Imports["example.com/partial"] = &Package{ID: "partial"}
	response := metadataResponse{
		Roots: []string{"root"}, Compiler: "gc", Arch: "386",
		Packages: []packageMetadata{
			{Package: root, Dir: "/app", Target: "/bin/app", ForTest: "example.com/root", Sizes: true,
				Module: &gopackages.Module{Path: "example.com/root", GoVersion: "1.20", GoMod: "/app/go.mod", Replace: &gopackages.Module{Path: "example.com/replacement", Dir: "/replacement"}}},
			{Package: dep, Imports: true},
		},
	}
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var decoded metadataResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	pkgs, err := decoded.packages()
	if err != nil || len(pkgs) != 1 {
		t.Fatalf("decode: %v / %v", err, pkgs)
	}
	pkg := pkgs[0]
	if !reflect.DeepEqual(pkg.Module, response.Packages[0].Module) || pkg.Dir != "/app" || pkg.Target != "/bin/app" || pkg.ForTest != "example.com/root" {
		t.Fatalf("supplemental metadata lost: %+v", pkg)
	}
	if pkg.Imports["example.com/dep"] != decoded.Packages[1].Package || pkg.TypesSizes != types.SizesFor("gc", "386") {
		t.Fatal("graph identity or exact gc sizes lost")
	}
	if decoded.Packages[1].Package.Imports == nil {
		t.Fatal("empty import map became nil")
	}
	if stub := pkg.Imports["example.com/partial"]; stub == nil || stub.ID != "partial" {
		t.Fatal("partial metadata lost its import stub")
	}
}

func TestMetadataDriverRequiresMarker(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, metadataDriverArg)
	cmd.Env = replaceEnvironment(os.Environ(), metadataDriverEnv, "")
	output, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "flag provided but not defined") {
		t.Fatalf("private argument alone entered the worker: %v / %s", err, output)
	}
	cmd = exec.Command(executable, metadataDriverArg)
	cmd.Env = replaceEnvironment(os.Environ(), metadataDriverEnv, "1")
	cmd.Stdin = strings.NewReader("{")
	output, err = cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "unexpected EOF") {
		t.Fatalf("invalid worker input: %v / %s", err, output)
	}
}

func TestDescribeSizesSupportedArchitectures(t *testing.T) {
	// Fail visibly if the Go SDK adds a gc architecture without updating the
	// transport. Read the SDK table rather than repeating the production list.
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(runtime.GOROOT(), "src", "go", "types", "sizes.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	ast.Inspect(file, func(node ast.Node) bool {
		value, ok := node.(*ast.ValueSpec)
		if !ok || len(value.Names) != 1 || value.Names[0].Name != "gcArchSizes" {
			return true
		}
		for _, element := range value.Values[0].(*ast.CompositeLit).Elts {
			key := element.(*ast.KeyValueExpr).Key.(*ast.BasicLit)
			arch, err := strconv.Unquote(key.Value)
			if err != nil {
				t.Fatal(err)
			}
			compiler, gotArch, std := describeSizes(types.SizesFor("gc", arch))
			if compiler != "gc" || gotArch != arch || std != nil {
				t.Errorf("gc/%s is not represented by the metadata transport", arch)
			}
			checked++
		}
		return false
	})
	if checked == 0 {
		t.Fatal("Go SDK gc architecture table not found")
	}
}
