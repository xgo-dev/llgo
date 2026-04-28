//go:build !llgo
// +build !llgo

package env

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestLLGoROOTWarnsOnceForDevelRoot(t *testing.T) {
	origLLGoRoot := os.Getenv("LLGO_ROOT")
	os.Setenv("LLGO_ROOT", "")
	t.Cleanup(func() { os.Setenv("LLGO_ROOT", origLLGoRoot) })

	oldWarned := llgoRootWarned
	llgoRootWarned = sync.Map{}
	t.Cleanup(func() { llgoRootWarned = oldWarned })

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	root1 := LLGoROOT()
	root2 := LLGoROOT()
	w.Close()
	os.Stderr = oldStderr
	defer r.Close()

	if root1 == "" || root2 == "" {
		t.Skip("LLGoROOT did not resolve a development root")
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	want := "WARNING: Using LLGO root for devel: " + root1
	if got := strings.Count(string(out), want); got != 1 {
		t.Fatalf("devel root warning count = %d, output %q", got, out)
	}
}

func TestGOROOT(t *testing.T) {
	// Test with GOROOT environment variable set
	t.Run("with GOROOT bad env", func(t *testing.T) {
		origGoRoot := os.Getenv("GOROOT")
		defer os.Setenv("GOROOT", origGoRoot)

		bad := "/custom/badgoroot"
		os.Setenv("GOROOT", bad)
		if got, err := GOROOT(); got == bad || err == nil {
			t.Fatal("bad GOROOT")
		}
	})

	// Test without GOROOT environment variable
	t.Run("without GOROOT env", func(t *testing.T) {
		origGoRoot := os.Getenv("GOROOT")
		defer os.Setenv("GOROOT", origGoRoot)

		os.Setenv("GOROOT", "")
		if got, _ := GOROOT(); got == "" {
			t.Error("GOROOT() should not return empty when using go env")
		}
	})
}

func TestGOROOTWithEnv(t *testing.T) {
	t.Run("with bad explicit env", func(t *testing.T) {
		env := appendEnv(os.Environ(), "GOROOT=/custom/badgoroot")
		if got, err := GOROOTWithEnv(env); got == "/custom/badgoroot" || err == nil {
			t.Fatal("GOROOTWithEnv should reject bad explicit GOROOT")
		}
	})

	t.Run("with explicit inherited env", func(t *testing.T) {
		got, err := GOROOTWithEnv(os.Environ())
		if err != nil {
			t.Fatal(err)
		}
		if got == "" {
			t.Fatal("GOROOTWithEnv should not return empty")
		}
	})
}

func TestGOVERSIONWithEnv(t *testing.T) {
	t.Run("with explicit inherited env", func(t *testing.T) {
		got, err := GOVERSIONWithEnv(os.Environ())
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(got, "go1.") {
			t.Fatalf("GOVERSIONWithEnv() = %q, want go1.x", got)
		}
	})

	t.Run("with go env failure", func(t *testing.T) {
		prependFakeGo(t, "echo bad goversion >&2\nexit 2\n")
		if got, err := GOVERSIONWithEnv(nil); err == nil || got != "" {
			t.Fatalf("GOVERSIONWithEnv() = %q, %v, want error", got, err)
		}
	})
}

func TestGOROOTAndGOVERSIONWithEnv(t *testing.T) {
	goroot, goversion, err := GOROOTAndGOVERSIONWithEnv(os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	if goroot == "" {
		t.Fatal("GOROOTAndGOVERSIONWithEnv should not return empty GOROOT")
	}
	if !strings.HasPrefix(goversion, "go1.") {
		t.Fatalf("GOROOTAndGOVERSIONWithEnv goversion = %q, want go1.x", goversion)
	}

	t.Run("with go env failure", func(t *testing.T) {
		prependFakeGo(t, "echo bad go env >&2\nexit 2\n")
		goroot, goversion, err := GOROOTAndGOVERSIONWithEnv(nil)
		if err == nil || goroot != "" || goversion != "" {
			t.Fatalf("GOROOTAndGOVERSIONWithEnv() = %q, %q, %v, want error", goroot, goversion, err)
		}
	})
}

func TestGoEnvWithEnvCachesSuccessfulResults(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake go shell script is Unix-only")
	}
	dir := t.TempDir()
	countFile := filepath.Join(dir, "count")
	goPath := filepath.Join(dir, "go")
	script := "#!/bin/sh\ncount=0\nif [ -f " + countFile + " ]; then count=$(cat " + countFile + "); fi\ncount=$((count + 1))\necho $count > " + countFile + "\necho /tmp/fake-goroot\necho go1.99.0\n"
	if err := os.WriteFile(goPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	for i := 0; i < 2; i++ {
		got, err := GoEnvWithEnv(nil, "GOROOT", "GOVERSION")
		if err != nil {
			t.Fatalf("GoEnvWithEnv call %d: %v", i, err)
		}
		if strings.Join(got, ",") != "/tmp/fake-goroot,go1.99.0" {
			t.Fatalf("GoEnvWithEnv call %d = %v", i, got)
		}
	}
	count, err := os.ReadFile(countFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(count)) != "1" {
		t.Fatalf("fake go invoked %s times, want 1", strings.TrimSpace(string(count)))
	}
}

func TestGoEnvWithEnvErrors(t *testing.T) {
	t.Run("without variables", func(t *testing.T) {
		if got, err := GoEnvWithEnv(nil); err == nil || got != nil {
			t.Fatalf("GoEnvWithEnv() = %v, %v, want error", got, err)
		}
	})

	t.Run("wrong output count", func(t *testing.T) {
		prependFakeGo(t, "echo only-one\n")
		if got, err := GoEnvWithEnv(nil, "GOROOT", "GOVERSION"); err == nil || got != nil {
			t.Fatalf("GoEnvWithEnv() = %v, %v, want output count error", got, err)
		}
	})
}

func TestLLGoRuntimeDir(t *testing.T) {
	// Test with valid LLGO_ROOT
	t.Run("with valid LLGO_ROOT", func(t *testing.T) {
		origLLGoRoot := os.Getenv("LLGO_ROOT")
		defer os.Setenv("LLGO_ROOT", origLLGoRoot)

		tmpDir := t.TempDir()
		runtimeDir := filepath.Join(tmpDir, "runtime")
		os.MkdirAll(runtimeDir, 0755)
		goModContent := []byte("module github.com/goplus/llgo/runtime\n")
		if err := os.WriteFile(filepath.Join(runtimeDir, "go.mod"), goModContent, 0644); err != nil {
			t.Fatal(err)
		}

		os.Setenv("LLGO_ROOT", tmpDir)
		expected := filepath.Join(tmpDir, LLGoRuntimePkgName)
		if got := LLGoRuntimeDir(); got != expected {
			t.Errorf("LLGoRuntimeDir() = %v, want %v", got, expected)
		}
	})

	// Test with invalid LLGO_ROOT
	t.Run("with invalid LLGO_ROOT", func(t *testing.T) {
		origLLGoRoot := os.Getenv("LLGO_ROOT")
		defer os.Setenv("LLGO_ROOT", origLLGoRoot)

		os.Setenv("LLGO_ROOT", "/nonexistent/path")
		wd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		runtimeDir := filepath.Join(wd, "../../runtime")
		if got := LLGoRuntimeDir(); got != runtimeDir {
			t.Errorf("LLGoRuntimeDir() = %v, want %v", got, runtimeDir)
		}
	})

	t.Run("devel runtime dir", func(t *testing.T) {
		origLLGoRoot := os.Getenv("LLGO_ROOT")
		defer os.Setenv("LLGO_ROOT", origLLGoRoot)

		os.Setenv("LLGO_ROOT", "")
		wd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		runtimeDir := filepath.Join(wd, "../../runtime")
		if got := LLGoRuntimeDir(); got != runtimeDir {
			t.Errorf("LLGoRuntimeDir() = %v, want %v", got, runtimeDir)
		}
	})
}

func TestLLGoROOT(t *testing.T) {
	// Test with valid LLGO_ROOT environment variable
	t.Run("with valid LLGO_ROOT env", func(t *testing.T) {
		origLLGoRoot := os.Getenv("LLGO_ROOT")
		defer os.Setenv("LLGO_ROOT", origLLGoRoot)

		tmpDir := t.TempDir()
		runtimeDir := filepath.Join(tmpDir, "runtime")
		os.MkdirAll(runtimeDir, 0755)
		goModContent := []byte("module github.com/goplus/llgo/runtime\n")
		if err := os.WriteFile(filepath.Join(runtimeDir, "go.mod"), goModContent, 0644); err != nil {
			t.Fatal(err)
		}

		os.Setenv("LLGO_ROOT", tmpDir)
		if got := LLGoROOT(); got != tmpDir {
			t.Errorf("LLGoROOT() = %v, want %v", got, tmpDir)
		}
	})

	// Test with invalid LLGO_ROOT environment variable
	t.Run("with invalid LLGO_ROOT env", func(t *testing.T) {
		origLLGoRoot := os.Getenv("LLGO_ROOT")
		defer os.Setenv("LLGO_ROOT", origLLGoRoot)

		os.Setenv("LLGO_ROOT", "/nonexistent/path")
		wd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		rootDir := filepath.Join(wd, "../..")
		if got := LLGoROOT(); got != rootDir {
			t.Errorf("LLGoROOT() = %v, want %v", got, rootDir)
		}
	})

	// Test with empty LLGO_ROOT environment variable
	t.Run("with empty LLGO_ROOT env", func(t *testing.T) {
		origLLGoRoot := os.Getenv("LLGO_ROOT")
		defer os.Setenv("LLGO_ROOT", origLLGoRoot)

		os.Setenv("LLGO_ROOT", "")
		// Result depends on executable path, just ensure it doesn't panic
		LLGoROOT()
	})
}

func TestIsLLGoRoot(t *testing.T) {
	// Test with empty root
	t.Run("empty root", func(t *testing.T) {
		if root, ok := isLLGoRoot(""); ok {
			t.Errorf("isLLGoRoot('') = %v, %v, want '', false", root, ok)
		}
	})

	// Test with invalid path
	t.Run("invalid path", func(t *testing.T) {
		if root, ok := isLLGoRoot(string([]byte{0})); ok {
			t.Errorf("isLLGoRoot(invalid) = %v, %v, want '', false", root, ok)
		}
	})

	// Test with non-existent path
	t.Run("non-existent path", func(t *testing.T) {
		if root, ok := isLLGoRoot("/nonexistent/path"); ok {
			t.Errorf("isLLGoRoot(nonexistent) = %v, %v, want '', false", root, ok)
		}
	})

	// Test with valid path but invalid go.mod
	t.Run("invalid go.mod", func(t *testing.T) {
		tmpDir := t.TempDir()
		runtimeDir := filepath.Join(tmpDir, "runtime")
		os.MkdirAll(runtimeDir, 0755)
		goModContent := []byte("module wrong/module/name\n")
		if err := os.WriteFile(filepath.Join(runtimeDir, "go.mod"), goModContent, 0644); err != nil {
			t.Fatal(err)
		}

		if root, ok := isLLGoRoot(tmpDir); ok {
			t.Errorf("isLLGoRoot(invalid_mod) = %v, %v, want '', false", root, ok)
		}
	})

	// Test with valid path and valid go.mod
	t.Run("valid path and go.mod", func(t *testing.T) {
		tmpDir := t.TempDir()
		runtimeDir := filepath.Join(tmpDir, "runtime")
		os.MkdirAll(runtimeDir, 0755)
		goModContent := []byte("module github.com/goplus/llgo/runtime\n")
		if err := os.WriteFile(filepath.Join(runtimeDir, "go.mod"), goModContent, 0644); err != nil {
			t.Fatal(err)
		}

		absPath, _ := filepath.Abs(tmpDir)
		if root, ok := isLLGoRoot(tmpDir); !ok || root != absPath {
			t.Errorf("isLLGoRoot(valid) = %v, %v, want %v, true", root, ok, absPath)
		}
	})
}

func appendEnv(base []string, overrides ...string) []string {
	out := append([]string(nil), base...)
	for _, override := range overrides {
		key, _, ok := strings.Cut(override, "=")
		if !ok {
			out = append(out, override)
			continue
		}
		replaced := false
		prefix := key + "="
		for i, entry := range out {
			if strings.HasPrefix(entry, prefix) {
				out[i] = override
				replaced = true
				break
			}
		}
		if !replaced {
			out = append(out, override)
		}
	}
	return out
}

func prependFakeGo(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake go shell script is Unix-only")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "go")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
