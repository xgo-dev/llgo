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
