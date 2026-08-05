//go:build !llgo

package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDebugTypeAssertReportsInitialPackageOnly(t *testing.T) {
	t.Setenv("LLGO_BUILD_CACHE", "off")

	sourcePath := filepath.Join(t.TempDir(), "typeassert.go")
	source := `package p

func assert(v any) int {
	return v.(int)
}
`
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	stderr, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = stderr
	t.Cleanup(func() { os.Stderr = oldStderr })

	conf := NewDefaultConf(ModeGen)
	conf.DebugTypeAssert = true
	conf.NoErrorColumn = true
	conf.Verbose = true
	pkgs, buildErr := Do([]string{sourcePath}, conf)

	os.Stderr = oldStderr
	if closeErr := stderr.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if buildErr != nil {
		t.Fatal(buildErr)
	}
	if len(pkgs) != 1 || pkgs[0].LPkg == nil {
		t.Fatalf("Do returned packages = %+v, want one compiled package", pkgs)
	}
	pkgs[0].LPkg.Prog.Dispose()

	got, err := os.ReadFile(stderr.Name())
	if err != nil {
		t.Fatal(err)
	}
	want := sourcePath + ":4: type assertion inlined\n"
	if !strings.Contains(string(got), want) {
		t.Fatalf("stderr does not contain %q:\n%s", want, got)
	}
	if count := strings.Count(string(got), "type assertion "); count != 1 {
		t.Fatalf("type assertion diagnostics = %d, want 1:\n%s", count, got)
	}
}
