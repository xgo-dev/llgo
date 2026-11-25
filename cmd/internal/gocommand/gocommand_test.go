package gocommand

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/mockable"
)

func TestParseBuildArgs(t *testing.T) {
	got, err := parseBuildArgs([]string{"-target=board", "-tags", "one,two one", "-x", "--", "-tags=literal"})
	if err != nil {
		t.Fatal(err)
	}
	want := buildQuery{
		target: "board",
		tags:   []string{"one", "two", "one"},
		goArgs: []string{"-x", "--", "-tags=literal"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseBuildArgs = %#v, want %#v", got, want)
	}
	for _, args := range [][]string{
		{"-target"},
		{"-target="},
		{"-target", ""},
		{"-target=", "-target=board"},
		{"-tags"},
	} {
		if _, err := parseBuildArgs(args); err == nil {
			t.Errorf("parseBuildArgs(%q) succeeded", args)
		}
	}
}

func TestBuildModuleMode(t *testing.T) {
	for _, flag := range []string{"-m", "-m=1", "-m=t", "-m=T", "-m=TRUE", "-m=true", "-m=True"} {
		args := []string{flag, "-json", "all"}
		inv, err := Build("list", args)
		if err != nil || !reflect.DeepEqual(inv.Args, args) {
			t.Errorf("module query %q = %q, %v", flag, inv.Args, err)
		}
	}
	for _, args := range [][]string{
		{"-m=0"},
		{"-m=f"},
		{"-m=FALSE"},
		{"-m=false"},
		{"-m=False"},
		{"-m", "-m=false"},
		{"--", "-m"},
		{"-tags", "-m"},
		{"-tags", "-m=true"},
		{"-m=invalid"},
	} {
		inv, err := Build("list", args)
		if err != nil {
			t.Fatal(err)
		}
		if len(inv.Args) == 0 || !strings.HasPrefix(inv.Args[0], "-tags=llgo,") {
			t.Errorf("package query %q lost LLGo defaults: %q", args, inv.Args)
		}
	}
	// Flags from other commands must not select list's module-query behavior.
	inv, err := Build("get", []string{"-m"})
	if err != nil || len(inv.Args) == 0 || !strings.HasPrefix(inv.Args[0], "-tags=llgo,") {
		t.Fatalf("get -m = %q, %v", inv.Args, err)
	}
}

func TestMergeTagsAndEnvironment(t *testing.T) {
	if got := mergeTags([]string{"llgo", "purego"}, []string{"board", "llgo"}, splitTags("user, purego")); !reflect.DeepEqual(got, []string{"llgo", "purego", "board", "user"}) {
		t.Fatalf("mergeTags = %q", got)
	}
	environ := replaceEnv([]string{"PATH=/bin", "GOOS=old"}, "GOOS", "linux", "GOARCH", "arm")
	if envValue(environ, "GOOS") != "linux" || envValue(environ, "GOARCH") != "arm" || envValue(environ, "PATH") != "/bin" {
		t.Fatalf("replaceEnv = %q", environ)
	}
}

func TestAddBuildTagsPreservesLeadingDirectoryFlag(t *testing.T) {
	for _, test := range []struct {
		args []string
		want []string
	}{
		{[]string{"."}, []string{"-tags=llgo", "."}},
		{[]string{"-C", "dir", "."}, []string{"-C", "dir", "-tags=llgo", "."}},
		{[]string{"-C=dir", "."}, []string{"-C=dir", "-tags=llgo", "."}},
	} {
		got, err := addBuildTags(test.args, "-tags=llgo")
		if err != nil || !reflect.DeepEqual(got, test.want) {
			t.Errorf("addBuildTags(%q) = %q, %v; want %q", test.args, got, err, test.want)
		}
	}
	if _, err := addBuildTags([]string{"-C"}, "-tags=llgo"); err == nil {
		t.Fatal("addBuildTags accepted -C without a value")
	}
}

func TestBuildTargetSelectionDoesNotCreateCache(t *testing.T) {
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
	cache := filepath.Join(t.TempDir(), "cache")
	t.Setenv("LLGO_ROOT", root)
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("LOCALAPPDATA", cache)

	inv, err := Build("list", []string{"-target", "board", "-tags=user,board", "."})
	if err != nil {
		t.Fatal(err)
	}
	if inv.Command != "list" || envValue(inv.Env, "GOOS") != "linux" || envValue(inv.Env, "GOARCH") != "arm" {
		t.Fatalf("target invocation = %#v", inv)
	}
	if len(inv.Args) == 0 || inv.Args[0] != "-tags=llgo,math_big_pure_go,purego,board,user" {
		t.Fatalf("target arguments = %q", inv.Args)
	}
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatalf("Build created cross-compilation cache: %v", err)
	}
}

func TestInvocation(t *testing.T) {
	var output bytes.Buffer
	inv := Invocation{Command: "env", Args: []string{"GOHOSTOS"}, Stdout: &output, Stderr: &output}
	if err := inv.Run(); err != nil {
		t.Fatalf("go env: %v, %s", err, &output)
	}
	if got := strings.TrimSpace(output.String()); got != runtime.GOOS {
		t.Fatalf("GOHOSTOS = %q, want %q", got, runtime.GOOS)
	}
}

func TestBuildAwareGenerateSelectsLLGoFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/gentest\n\ngo 1.20\n"), 0644); err != nil {
		t.Fatal(err)
	}
	source := "//go:build llgo\n\npackage gentest\n\n//go:generate go env GOOS\n"
	if err := os.WriteFile(filepath.Join(dir, "generate.go"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	inv, err := Build("generate", []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	inv.Stdout, inv.Stderr = &output, &output
	if err := inv.Run(); err != nil {
		t.Fatalf("go generate: %v, %s", err, &output)
	}
	if got := strings.TrimSpace(output.String()); got != runtime.GOOS {
		t.Fatalf("generated GOOS = %q, want %q", got, runtime.GOOS)
	}
}

func TestTransparentFmtIncludesIgnoredGoFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/fmttest\n\ngo 1.27\n"), 0644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ignored.go")
	if err := os.WriteFile(path, []byte("//go:build never_enabled\n\npackage fmttest\n\nfunc f( ){println(1)}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	var output bytes.Buffer
	if err := (Invocation{Command: "fmt", Args: []string{"."}, Stdout: &output, Stderr: &output}).Run(); err != nil {
		t.Fatalf("go fmt: %v, %s", err, &output)
	}
	formatted, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(formatted), "func f() { println(1) }") {
		t.Fatalf("ignored file was not formatted:\n%s", formatted)
	}
}

func TestTransparentWork(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	var output bytes.Buffer
	if err := (Invocation{Command: "work", Args: []string{"init"}, Stdout: &output, Stderr: &output}).Run(); err != nil {
		t.Fatalf("go work init: %v, %s", err, &output)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.work")); err != nil {
		t.Fatalf("go work init did not create go.work: %v", err)
	}
}

func TestExit(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	err := (Invocation{Command: "env"}).Run()
	if err == nil {
		t.Fatal("Invocation succeeded without Go")
	}
	mockable.EnableMock()
	defer mockable.DisableMock()
	defer func() {
		if got := recover(); got != "exit" || mockable.ExitCode() != 1 {
			t.Errorf("exit = %v, %d", got, mockable.ExitCode())
		}
	}()
	Exit("env", err)
}
