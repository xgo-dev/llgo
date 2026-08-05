package env

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestExpandEnvToArgsWithUsesExplicitEnvironment(t *testing.T) {
	t.Setenv("LLGO_ENV_TEST", "ambient")
	got := ExpandEnvToArgsWith("$LLGO_ENV_TEST", "", []string{"LLGO_ENV_TEST=request"})
	if want := []string{"request"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpandEnvToArgsWith = %q, want %q", got, want)
	}
}

func TestExpandEnvUsesProcessEnvironment(t *testing.T) {
	t.Setenv("LLGO_ENV_TEST", "ambient")
	if got := ExpandEnv("$LLGO_ENV_TEST"); got != "ambient" {
		t.Fatalf("ExpandEnv = %q, want %q", got, "ambient")
	}
	if got := ExpandEnvToArgs("$LLGO_ENV_TEST"); !reflect.DeepEqual(got, []string{"ambient"}) {
		t.Fatalf("ExpandEnvToArgs = %q, want %q", got, []string{"ambient"})
	}
	if got := ExpandEnvToArgs(""); got != nil {
		t.Fatalf("ExpandEnvToArgs(empty) = %q, want nil", got)
	}
}

func TestExpandEnvToArgsWithConfiguresSubprocess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-only")
	}
	dir := t.TempDir()
	tool := filepath.Join(dir, "pkg-config")
	script := "#!/bin/sh\nprintf '%s' \"-L$LLGO_ENV_TEST -I$PWD\"\n"
	if err := os.WriteFile(tool, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	got := ExpandEnvToArgsWith(
		"$(pkg-config --libs fixture)",
		dir,
		[]string{"PATH=" + dir, "LLGO_ENV_TEST=request"},
	)
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-Lrequest", "-I" + resolvedDir}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExpandEnvToArgsWith = %q, want %q", got, want)
	}
}

func TestLookPathInEnvironmentBoundaries(t *testing.T) {
	dir := t.TempDir()
	tool := filepath.Join(dir, "fixture-tool")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := lookPathInEnvironment("fixture-tool", dir, []string{"PATH=" + string(os.PathListSeparator)}); got != tool {
		t.Fatalf("lookPathInEnvironment with empty entry = %q, want %q", got, tool)
	}
	if got := lookPathInEnvironment(filepath.Join("bin", "tool"), dir, nil); got != filepath.Join("bin", "tool") {
		t.Fatalf("lookPathInEnvironment with separator = %q", got)
	}
	if got := lookPathInEnvironment("missing-tool", dir, []string{"PATH=" + t.TempDir()}); got != "missing-tool" {
		t.Fatalf("lookPathInEnvironment missing tool = %q", got)
	}
	if got := ExpandEnvToArgsWith("$LLGO_ENV_MISSING", dir, []string{"PATH=" + dir}); got != nil {
		t.Fatalf("missing explicit environment variable = %q, want nil", got)
	}
}
