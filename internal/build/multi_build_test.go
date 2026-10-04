package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
)

func TestPrepareBuildOutputEdges(t *testing.T) {
	root := t.TempDir()
	mainPackages := []*packages.Package{{Name: "main"}}
	if got, err := prepareBuildOutput(root, "", false, nil); err != nil || got != "" {
		t.Fatalf("empty output = %q, %v", got, err)
	}

	blocking := filepath.Join(root, "blocking")
	if err := os.WriteFile(blocking, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareBuildOutput(root, filepath.Join("blocking", "child")+string(os.PathSeparator), false, mainPackages); err == nil ||
		!strings.Contains(err.Error(), "create build output directory") {
		t.Fatalf("blocked output directory error = %v", err)
	}

	if filepath.Separator != '\\' {
		if got, err := prepareBuildOutput(root, `\`, false, mainPackages); err != nil || got != root {
			t.Fatalf("root backslash output = %q, %v; want %q", got, err, root)
		}
		want := filepath.Join(root, "nested")
		if got, err := prepareBuildOutput(root, `nested\`, false, mainPackages); err != nil || got != want {
			t.Fatalf("nested backslash output = %q, %v; want %q", got, err, want)
		}
	}
}

func TestDefaultExecutableName(t *testing.T) {
	for _, test := range []struct {
		path string
		want string
	}{
		{"example.com/tool", "tool"},
		{"example.com/tool/v2", "tool"},
		{"example.com/tool/v0", "v0"},
		{"example.com/tool/v1", "v1"},
		{"example.com/tool/vnext", "vnext"},
	} {
		if got := defaultExecutableName(test.path); got != test.want {
			t.Errorf("defaultExecutableName(%q) = %q, want %q", test.path, got, test.want)
		}
	}
}

func TestLinkInitialPackagePropagatesOutputCreationError(t *testing.T) {
	root := t.TempDir()
	blocking := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocking, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, blocking)
	}
	ctx := &context{initial: []*packages.Package{{}, {}}}
	conf := multiBuildConfig()
	conf.Mode = ModeRun
	_, err := linkInitialPackage(ctx, &packages.Package{PkgPath: "example.com/tool"}, nil, conf, false, false)
	if err == nil {
		t.Fatalf("temporary output creation error = %v", err)
	}
	fallback := newMultiBuildFallback(multiBuildConfig(), []*packages.Package{{
		Name: "main", PkgPath: "example.com/tool",
	}}, root, "")
	if _, err := fallback.run(); err == nil || !strings.Contains(err.Error(), "create temporary output") {
		t.Fatalf("fallback temporary output creation error = %v", err)
	}
}

func TestModeBuildMultiplePackagesMatchesGoOutputContract(t *testing.T) {
	root := writeMultiBuildModule(t, map[string]string{
		"cmd/first/main.go":  mainSource("first"),
		"cmd/second/main.go": mainSource("second"),
		"cmd/a/same/main.go": mainSource("same-a"),
		"cmd/b/same/main.go": mainSource("same-b"),
		"cmd/badlink/main.go": `package main
import _ "unsafe"
//go:linkname missing C.llgo_multi_build_missing_symbol
func missing()
func main() { missing() }
`,
		"lib/lib.go": "package lib\nfunc Value() int { return 1 }\n",
	})

	conf := multiBuildConfig()
	if _, err := Build(Invocation{Args: []string{"./..."}, Config: conf, Dir: root}); err == nil ||
		!strings.Contains(err.Error(), "example.com/multibuild/cmd/badlink") {
		t.Fatalf("multi-package build link error = %v", err)
	}
	for _, name := range []string{"first", "second", "same", "badlink"} {
		if _, err := os.Stat(filepath.Join(root, name+conf.AppExt)); !os.IsNotExist(err) {
			t.Fatalf("check-only build left output %q: %v", name, err)
		}
	}

	out := filepath.Join(root, "out")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	withOutput := multiBuildConfig()
	withOutput.OutFile = out
	_, err := Build(Invocation{Args: []string{"./..."}, Config: withOutput, Dir: root})
	if err == nil {
		t.Fatal("directory build unexpectedly linked the missing symbol")
	}
	if count := strings.Count(err.Error(), "example.com/multibuild/cmd/badlink:"); count != 1 {
		t.Fatalf("badlink error package prefix count = %d, want 1: %v", count, err)
	}
	for name, want := range map[string]string{"first": "first", "second": "second"} {
		assertBuiltProgram(t, filepath.Join(out, name+withOutput.AppExt), want)
	}
	// cmd/go also accepts colliding DefaultExecName values for a directory
	// output. One executable remains, but the winning package is unspecified.
	if got := runBuiltProgram(t, filepath.Join(out, "same"+withOutput.AppExt)); got != "same-a" && got != "same-b" {
		t.Fatalf("same-basename output = %q, want either main package", got)
	}
	if _, err := os.Stat(filepath.Join(out, "lib"+withOutput.AppExt)); !os.IsNotExist(err) {
		t.Fatalf("non-main package produced output: %v", err)
	}

	fileOutput := multiBuildConfig()
	fileOutput.OutFile = filepath.Join(root, "not-a-directory")
	if _, err := Build(Invocation{Args: []string{"./cmd/first", "./cmd/second"}, Config: fileOutput, Dir: root}); err == nil ||
		!strings.Contains(err.Error(), "cannot write multiple packages to non-directory") {
		t.Fatalf("multi-package -o file error = %v", err)
	}

	archive := multiBuildConfig()
	archive.BuildMode = BuildModeCArchive
	if _, err := Build(Invocation{Args: []string{"./cmd/first", "./cmd/second"}, Config: archive, Dir: root}); err == nil ||
		!strings.Contains(err.Error(), "-buildmode=c-archive requires exactly one main package") {
		t.Fatalf("multi-package c-archive error = %v", err)
	}
	singleArchive := multiBuildConfig()
	singleArchive.BuildMode = BuildModeCArchive
	singleArchive.OutFile = filepath.Join(root, "first.a")
	if _, err := Build(Invocation{Args: []string{"./cmd/first"}, Config: singleArchive, Dir: root}); err != nil {
		t.Fatalf("single-package c-archive build: %v", err)
	}
	for _, artifact := range []string{singleArchive.OutFile, strings.TrimSuffix(singleArchive.OutFile, ".a") + ".h"} {
		if _, err := os.Stat(artifact); err != nil {
			t.Fatalf("single-package c-archive artifact %q: %v", artifact, err)
		}
	}

	trailingDir := filepath.Join(root, "created") + string(os.PathSeparator)
	trailing := multiBuildConfig()
	trailing.OutFile = trailingDir
	if _, err := Build(Invocation{Args: []string{"./cmd/first", "./cmd/second"}, Config: trailing, Dir: root}); err != nil {
		t.Fatalf("trailing-directory build: %v", err)
	}
	assertBuiltProgram(t, filepath.Join(trailingDir, "first"+trailing.AppExt), "first")
	assertBuiltProgram(t, filepath.Join(trailingDir, "second"+trailing.AppExt), "second")
}

func TestRemoveTemporaryBuildOutputs(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "program.elf")
	base := strings.TrimSuffix(output, filepath.Ext(output))
	artifacts := []string{
		output, pclnSidecarPath(output), base + ".bin", base + ".hex",
		base + ".img", base + ".uf2", base + ".zip",
	}
	for _, artifact := range artifacts {
		if err := os.WriteFile(artifact, []byte("generated"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	removeTemporaryBuildOutputs(output)
	for _, artifact := range artifacts {
		if _, err := os.Stat(artifact); !os.IsNotExist(err) {
			t.Errorf("temporary artifact %q remains: %v", artifact, err)
		}
	}
}

func TestModeBuildMultiplePackagesRecoversIndependentRoots(t *testing.T) {
	root := writeMultiBuildModule(t, map[string]string{
		"cmd/good/main.go": mainSource("good"),
		"cmd/bad/main.go":  "package main\nfunc main() { _ = undefinedName }\n",
		"cmd/badlink/main.go": `package main
import _ "unsafe"
//go:linkname missing C.llgo_multi_build_fallback_missing_symbol
func missing()
func main() { missing() }
`,
	})
	out := filepath.Join(root, "out")
	conf := multiBuildConfig()
	conf.OutFile = out + string(os.PathSeparator)
	_, err := Build(Invocation{Args: []string{"./..."}, Config: conf, Dir: root})
	if err == nil || !strings.Contains(err.Error(), "example.com/multibuild/cmd/bad:") ||
		!strings.Contains(err.Error(), "example.com/multibuild/cmd/badlink:") {
		t.Fatalf("multi-package compile error = %v", err)
	}
	assertBuiltProgram(t, filepath.Join(out, "good"+conf.AppExt), "good")

	checkOnly := multiBuildConfig()
	_, checkErr := Build(Invocation{Args: []string{"./..."}, Config: checkOnly, Dir: root})
	if checkErr == nil || !strings.Contains(checkErr.Error(), "example.com/multibuild/cmd/bad:") ||
		!strings.Contains(checkErr.Error(), "example.com/multibuild/cmd/badlink:") {
		t.Fatalf("check-only multi-package compile error = %v", checkErr)
	}
	for _, name := range []string{"good", "badlink"} {
		if _, err := os.Stat(filepath.Join(root, name+checkOnly.AppExt)); !os.IsNotExist(err) {
			t.Fatalf("check-only fallback left output %q: %v", name, err)
		}
	}
}

func TestModeBuildOutputDirectoryRequiresMain(t *testing.T) {
	root := writeMultiBuildModule(t, map[string]string{
		"one/one.go": "package one\n",
		"two/two.go": "package two\n",
	})
	for _, args := range [][]string{{"./..."}, {"./one"}} {
		conf := multiBuildConfig()
		conf.OutFile = filepath.Join(root, "out") + string(os.PathSeparator)
		if _, err := Build(Invocation{Args: args, Config: conf, Dir: root}); err == nil ||
			!strings.Contains(err.Error(), "no main packages to build") {
			t.Fatalf("Build(%v) non-main directory output error = %v", args, err)
		}
	}
}

func TestModeBuildSinglePackageAcceptsOutputDirectory(t *testing.T) {
	root := writeMultiBuildModule(t, map[string]string{"cmd/only/main.go": mainSource("only")})
	out := filepath.Join(root, "out")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	conf := multiBuildConfig()
	conf.OutFile = out
	if _, err := Build(Invocation{Args: []string{"./cmd/only"}, Config: conf, Dir: root}); err != nil {
		t.Fatal(err)
	}
	assertBuiltProgram(t, filepath.Join(out, "only"+conf.AppExt), "only")
}

func TestModeBuildRejectsInvalidSizeFormat(t *testing.T) {
	conf := multiBuildConfig()
	conf.SizeReport = true
	conf.SizeFormat = "invalid"
	if _, err := Build(Invocation{Config: conf, Dir: t.TempDir()}); err == nil ||
		!strings.Contains(err.Error(), "invalid size format") {
		t.Fatalf("size report configuration error = %v", err)
	}
}

func multiBuildConfig() *Config {
	conf := NewDefaultConf(ModeBuild)
	conf.BuildParallelism = 2
	conf.PCLNMode = PCLNNone
	conf.PCLNModeSet = true
	return conf
}

func writeMultiBuildModule(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	files["go.mod"] = "module example.com/multibuild\n\ngo 1.25\n"
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func mainSource(output string) string {
	return "package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"" + output + "\") }\n"
}

func assertBuiltProgram(t *testing.T, path, want string) {
	t.Helper()
	if got := runBuiltProgram(t, path); got != want {
		t.Fatalf("%s output = %q, want %q", path, got, want)
	}
}

func runBuiltProgram(t *testing.T, path string) string {
	t.Helper()
	if runtime.GOOS == "windows" && filepath.Ext(path) == "" {
		path += ".exe"
	}
	out, err := exec.Command(path).CombinedOutput()
	if err != nil {
		t.Fatalf("run %s: %v\n%s", path, err, out)
	}
	return strings.TrimSpace(string(out))
}
