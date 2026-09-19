//go:build !llgo

package list

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/mockable"
)

func TestParseArgs(t *testing.T) {
	got, err := parseArgs([]string{"-target=board", "-tags", "one,two one", "-export=false", "-json=ImportPath,Name", "--", "-package"})
	if err != nil {
		t.Fatal(err)
	}
	want := listQuery{
		target:    "board",
		targetSet: true,
		tags:      []string{"one", "two", "one"},
		goArgs:    []string{"-export=false", "-json=ImportPath,Name", "--", "-package"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseArgs = %#v, want %#v", got, want)
	}
	for _, args := range [][]string{{"-target"}, {"-target="}, {"-tags"}} {
		if _, err := parseArgs(args); err == nil {
			t.Errorf("parseArgs(%q) succeeded", args)
		}
	}
	for _, flag := range []string{"-m", "-m=1", "-m=t", "-m=T", "-m=TRUE", "-m=true", "-m=True"} {
		module, err := parseArgs([]string{flag, "-json", "all"})
		if err != nil || !module.moduleMode {
			t.Errorf("module query %q = %#v, %v", flag, module, err)
		}
	}
	for _, flag := range []string{"-m=0", "-m=f", "-m=FALSE", "-m=false", "-m=False"} {
		module, err := parseArgs([]string{flag, "-json", "all"})
		if err != nil || module.moduleMode {
			t.Errorf("package query %q = %#v, %v", flag, module, err)
		}
	}
}

func TestMergeTagsAndEnvironment(t *testing.T) {
	if got := mergeTags([]string{"llgo", "purego"}, []string{"board", "llgo"}, splitTags("user, purego")); !reflect.DeepEqual(got, []string{"llgo", "purego", "board", "user"}) {
		t.Fatalf("mergeTags = %q", got)
	}
	environ := replaceEnv([]string{"PATH=/bin", "GOOS=old"}, "GOOS", "linux", "GOARCH", "arm")
	if !slicesContain(environ, "GOOS=linux") || !slicesContain(environ, "GOARCH=arm") || !slicesContain(environ, "PATH=/bin") {
		t.Fatalf("replaceEnv = %q", environ)
	}
	query := listQuery{moduleMode: true, tags: []string{"user", "board"}}
	if got := effectiveTags(query, "arm", []string{"board", "target"}); !reflect.DeepEqual(got, []string{"board", "target", "user"}) {
		t.Fatalf("module target tags = %q", got)
	}
}

func TestListWailsSizesQuery(t *testing.T) {
	var output bytes.Buffer
	args := []string{"-f", "{{context.GOARCH}} {{context.Compiler}}", "--", "unsafe"}
	if err := run(args, nil, &output, &output); err != nil {
		t.Fatalf("list %q: %v, %s", args, err, &output)
	}
	wantArch := os.Getenv("GOARCH")
	if wantArch == "" {
		wantArch = runtime.GOARCH
	}
	if got, want := strings.TrimSpace(output.String()), wantArch+" gc"; got != want {
		t.Fatalf("Wails sizes query = %q, want %q", got, want)
	}
}

func TestListGoPackagesQuery(t *testing.T) {
	var output bytes.Buffer
	args := []string{"-e", "-json=ImportPath,Name,Export,GoFiles,CompiledGoFiles", "-compiled=true", "-test=false", "-export=true", "-deps=false", "-find=false", "-pgo=off", "--", "errors"}
	if err := run(args, nil, &output, &output); err != nil {
		t.Fatalf("go/packages list query: %v, %s", err, &output)
	}
	var result struct {
		ImportPath string
		Name       string
		Export     string
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.ImportPath != "errors" || result.Name != "errors" || result.Export == "" {
		t.Fatalf("go/packages result = %#v", result)
	}
}

func TestListModuleQueryPreservesUserTags(t *testing.T) {
	var output bytes.Buffer
	args := []string{"-m", "-tags=llgo_test_tag", "-f", "{{.Path}}"}
	if err := run(args, nil, &output, &output); err != nil {
		t.Fatalf("module list %q: %v, %s", args, err, &output)
	}
	if strings.TrimSpace(output.String()) != "github.com/xgo-dev/llgo" {
		t.Fatalf("module list = %q", &output)
	}
}

func TestListSourceSelection(t *testing.T) {
	module := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(module, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/listtest\n\ngo 1.27\n")
	write("llgo.go", "//go:build llgo\n\npackage listtest\n")
	write("goonly.go", "//go:build !llgo\n\npackage listtest\n")
	write("user.go", "//go:build usertag\n\npackage listtest\n")
	t.Chdir(module)

	var output bytes.Buffer
	if err := run([]string{"-tags=usertag", "-f", "{{join .GoFiles \",\"}}", "."}, nil, &output, &output); err != nil {
		t.Fatalf("source selection: %v, %s", err, &output)
	}
	files := strings.Split(strings.TrimSpace(output.String()), ",")
	if !slicesContain(files, "llgo.go") || !slicesContain(files, "user.go") || slicesContain(files, "goonly.go") {
		t.Fatalf("selected files = %q", files)
	}
}

func TestListTargetSelectionDoesNotCreateCache(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("runtime/go.mod", "module github.com/xgo-dev/llgo/runtime\n")
	write("targets/board.json", `{"goos":"linux","goarch":"arm","build-tags":["board"]}`)
	module := t.TempDir()
	if err := os.WriteFile(filepath.Join(module, "go.mod"), []byte("module example.com/board\n\ngo 1.27\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(module, "board.go"), []byte("//go:build board\n\npackage board\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(t.TempDir(), "cache")
	t.Setenv("LLGO_ROOT", root)
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("LOCALAPPDATA", cache)
	t.Chdir(module)
	var output bytes.Buffer
	if err := run([]string{"-target", "board", "-f", "{{context.GOOS}}/{{context.GOARCH}} {{join .GoFiles \",\"}}", "."}, nil, &output, &output); err != nil {
		t.Fatalf("target list: %v, %s", err, &output)
	}
	if got := strings.TrimSpace(output.String()); got != "linux/arm board.go" {
		t.Fatalf("target selection = %q", got)
	}
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatalf("list created cross-compilation cache: %v", err)
	}
}

func TestRunCmdFailure(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	mockable.EnableMock()
	defer mockable.DisableMock()
	defer func() {
		if got := recover(); got != "exit" || mockable.ExitCode() != 1 {
			t.Errorf("exit = %v, %d", got, mockable.ExitCode())
		}
	}()
	Cmd.Run(Cmd, nil)
}

func TestListErrors(t *testing.T) {
	var output bytes.Buffer
	if err := run([]string{"-target", "missing-llgo-target", "unsafe"}, nil, &output, &output); err == nil || !strings.Contains(err.Error(), "missing-llgo-target") {
		t.Fatalf("missing target error = %v", err)
	}
	if err := run([]string{"-definitely-not-a-go-list-flag"}, nil, &output, &output); err == nil {
		t.Fatal("invalid Go list flag succeeded")
	}
}

func slicesContain(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
