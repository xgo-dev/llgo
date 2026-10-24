package compile

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/xgo-dev/llgo/xtool/nm"
)

func TestIsCompiled(t *testing.T) {
	t.Run("IsCompiled Not Exists", func(t *testing.T) {
		cfg := CompileConfig{
			Groups: []CompileGroup{
				{
					OutputFileName: "fakefile1.a",
				},
			},
		}

		if cfg.Groups[0].IsCompiled(".") {
			t.Errorf("unexpected result: should false")
		}
	})
	t.Run("IsCompiled Exists", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test*.a")
		if err != nil {
			t.Error(err)
			return
		}
		defer os.Remove(tmpFile.Name())
		cfg := CompileConfig{
			Groups: []CompileGroup{
				{
					OutputFileName: tmpFile.Name(),
				},
			},
		}

		if !cfg.Groups[0].IsCompiled(filepath.Dir(tmpFile.Name())) {
			t.Errorf("unexpected result: should true")
		}
	})
}

func TestCompile(t *testing.T) {
	t.Run("Skip compile", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-compile*")
		if err != nil {
			t.Error(err)
			return
		}
		defer os.RemoveAll(tmpDir)

		tmpFile, err := os.CreateTemp(tmpDir, "test*.a")
		if err != nil {
			t.Error(err)
			return
		}
		group := CompileGroup{
			OutputFileName: tmpFile.Name(),
		}
		err = group.Compile(tmpDir, CompileOptions{
			CC:     "clang",
			Linker: "lld",
		})
		if err != nil {
			t.Errorf("unexpected result: should nil: %v", err)
		}
	})

	t.Run("TmpDir Fail", func(t *testing.T) {
		tmpDir := filepath.Join(t.TempDir(), "test-compile")
		if err := os.Mkdir(tmpDir, 0o755); err != nil {
			t.Fatal(err)
		}
		badTempRoot := filepath.Join(t.TempDir(), "not-a-directory")
		if err := os.WriteFile(badTempRoot, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		// A mode-000 directory is still writable on Windows and by privileged
		// Unix users. A regular file is never a valid temporary-directory root.
		t.Setenv("TMPDIR", badTempRoot)
		t.Setenv("TMP", badTempRoot)
		t.Setenv("TEMP", badTempRoot)

		group := CompileGroup{
			OutputFileName: "nop.a",
		}
		err := group.Compile(tmpDir, CompileOptions{
			CC:     "clang",
			Linker: "lld",
		})
		if err == nil {
			t.Errorf("unexpected result: should not nil")
		}
	})

	t.Run("Compile", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-compile*")
		if err != nil {
			t.Error(err)
			return
		}
		defer os.RemoveAll(tmpDir)

		tmpFile, err := os.CreateTemp(tmpDir, "test*.c")
		if err != nil {
			t.Error(err)
			return
		}

		_, err = tmpFile.Write([]byte(`#include <math.h>
			void Foo() {
				double x = 2.0;
				double y = sqrt(x);
				(void) y ;
			}
			`))
		if err != nil {
			t.Error(err)
			return
		}

		group := CompileGroup{
			OutputFileName: "nop.a",
			Files:          []string{tmpFile.Name()},
		}
		err = group.Compile(tmpDir, CompileOptions{
			CC:      "clang",
			Linker:  "lld",
			CCFLAGS: []string{"-nostdinc"},
		})
		if err == nil {
			t.Errorf("unexpected result: should not nil")
		}
		err = group.Compile(tmpDir, CompileOptions{
			CC:     "clang",
			Linker: "lld",
		})
		if err != nil {
			t.Errorf("unexpected result: should not nil")
		}
		if _, err := os.Stat(filepath.Join(tmpDir, "nop.a")); os.IsNotExist(err) {
			t.Error("unexpected result: compiled nop.a not found")
			return
		}

		items, err := nm.New("").List(filepath.Join(tmpDir, "nop.a"))
		if err != nil {
			t.Error(err)
			return
		}

		want := "Foo"
		found := false
	loop:
		for _, item := range items {
			for _, sym := range item.Symbols {
				if strings.Contains(sym.Name, want) {
					found = true
					break loop
				}
			}
		}
		if !found {
			t.Errorf("cannot find symbol Foo")
		}
	})

	t.Run("Compile Asm", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-compile*")
		if err != nil {
			t.Error(err)
			return
		}
		defer os.RemoveAll(tmpDir)
		tmpFile, err := os.CreateTemp(tmpDir, "test*.S")
		if err != nil {
			t.Error(err)
			return
		}
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write([]byte(`
	.text
	.globl	_test
			`))
		if err != nil {
			t.Error(err)
			return
		}

		group := CompileGroup{
			OutputFileName: "nop.a",
			Files:          []string{tmpFile.Name()},
		}
		err = group.Compile(tmpDir, CompileOptions{
			CC:      "clang",
			Linker:  "lld",
			CCFLAGS: []string{"--target=x86_64-linux-gnu"},
		})
		if err != nil {
			t.Errorf("unexpected result: should nil %v", err)
		}
	})
}

func TestObjectFilePattern(t *testing.T) {
	if got, want := objectFilePattern(filepath.Join("source tree", "foo:bar.c")), "foo-bar.c-*.o"; got != want {
		t.Fatalf("objectFilePattern = %q, want %q", got, want)
	}
}

func TestWriteArchiveResponseFile(t *testing.T) {
	t.Run("quoted object paths", func(t *testing.T) {
		dir := t.TempDir()
		objFiles := []string{
			filepath.Join(dir, "first.o"),
			filepath.Join(dir, "directory with spaces", "second.o"),
		}
		responseFile, err := writeArchiveResponseFile(dir, objFiles)
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(responseFile)
		if err != nil {
			t.Fatal(err)
		}
		want := fmt.Sprintf("\"%s\"\n\"%s\"\n", filepath.ToSlash(objFiles[0]), filepath.ToSlash(objFiles[1]))
		if string(got) != want {
			t.Fatalf("response file = %q, want %q", got, want)
		}
	})

	t.Run("write error", func(t *testing.T) {
		missingDir := filepath.Join(t.TempDir(), "missing")
		if _, err := writeArchiveResponseFile(missingDir, []string{"object.o"}); err == nil {
			t.Fatal("writeArchiveResponseFile succeeded in a missing directory")
		}
	})
}

func TestCompileArchiveErrorIncludesContext(t *testing.T) {
	outputDir := t.TempDir()
	group := CompileGroup{OutputFileName: "broken.a"}
	err := group.Compile(outputDir, CompileOptions{
		CC: filepath.Join(t.TempDir(), "missing-clang"),
	})
	if err == nil {
		t.Fatal("Compile succeeded without llvm-ar")
	}
	want := "llvm-ar rcs " + filepath.Join(outputDir, group.OutputFileName)
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Compile error = %q, want context %q", err, want)
	}
}

func TestCompileFilesConcurrency(t *testing.T) {
	for _, tc := range []struct {
		procs, files, want int
	}{
		{8, 0, 0},
		{8, 1, 1},
		{1, 9, 1},
		{2, 9, 2},
		{8, 9, 4},
	} {
		t.Run(fmt.Sprintf("procs%d-files%d", tc.procs, tc.files), func(t *testing.T) {
			previous := runtime.GOMAXPROCS(tc.procs)
			t.Cleanup(func() { runtime.GOMAXPROCS(previous) })
			synctest.Test(t, func(t *testing.T) {
				started := make(chan int, tc.files)
				release := make(chan struct{})
				done := make(chan error, 1)
				go func() {
					done <- compileFiles(tc.files, func(index int) error {
						started <- index
						<-release
						return nil
					})
				}()
				// Every started compile is now blocked on release. This checks
				// the actual concurrency without sleeps or wall-clock thresholds.
				synctest.Wait()
				if got := len(started); got != tc.want {
					t.Errorf("concurrent compiles = %d, want %d", got, tc.want)
				}
				close(release)
				if err := <-done; err != nil {
					t.Fatal(err)
				}
				close(started)
				seen := make(map[int]bool)
				for index := range started {
					if index < 0 || index >= tc.files || seen[index] {
						t.Fatalf("unexpected or repeated source index %d", index)
					}
					seen[index] = true
				}
				if len(seen) != tc.files {
					t.Fatalf("compiled %d files, want %d", len(seen), tc.files)
				}
			})
		})
	}
}

func TestCompileFilesFailureWaitsForActiveCompiles(t *testing.T) {
	previous := runtime.GOMAXPROCS(4)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })
	synctest.Test(t, func(t *testing.T) {
		started := make(chan int, 20)
		finished := make(chan int, 20)
		fail := make(chan struct{})
		release := make(chan struct{})
		done := make(chan error, 1)
		wantErr := errors.New("compiler failed")
		go func() {
			done <- compileFiles(20, func(index int) error {
				started <- index
				if index == 0 {
					<-fail
					finished <- index
					return wantErr
				}
				<-release
				finished <- index
				return nil
			})
		}()
		synctest.Wait()
		close(fail)
		synctest.Wait()
		var err error
		returned := false
		select {
		case err = <-done:
			returned = true
			t.Error("returned while compiler processes were still active")
		default:
		}
		if len(started) != 4 || len(finished) != 1 {
			t.Errorf("before releasing active compiles: started=%d finished=%d", len(started), len(finished))
		}
		close(release)
		if !returned {
			err = <-done
		}
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
		if len(started) != 4 || len(finished) != 4 {
			t.Fatalf("started=%d finished=%d, want only the four active compiles", len(started), len(finished))
		}
	})
}

func TestCompileArchivePreservesSourceOrderAndDuplicateBasenames(t *testing.T) {
	group := CompileGroup{OutputFileName: "ordered.a"}
	sourceRoot := t.TempDir()
	for i := range 4 {
		dir := filepath.Join(sourceRoot, fmt.Sprintf("source %d", 3-i))
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(dir, "same.c")
		if err := os.WriteFile(file, fmt.Appendf(nil, "int Foo%d(void) { return %d; }\n", i, i), 0o644); err != nil {
			t.Fatal(err)
		}
		group.Files = append(group.Files, file)
	}
	outputDir := t.TempDir()
	if err := group.Compile(outputDir, CompileOptions{CC: "clang"}); err != nil {
		t.Fatal(err)
	}
	items, err := nm.New("llvm-nm").List(filepath.Join(outputDir, group.OutputFileName))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != len(group.Files) {
		t.Fatalf("archive members = %d, want %d", len(items), len(group.Files))
	}
	for i, item := range items {
		want := fmt.Sprintf("Foo%d", i)
		found := false
		for _, symbol := range item.Symbols {
			if strings.TrimPrefix(symbol.Name, "_") == want {
				found = true
			}
		}
		if !found {
			t.Errorf("archive member %d (%s) does not contain %s", i, item.File, want)
		}
	}
	mainFile := filepath.Join(sourceRoot, "main.c")
	if err := os.WriteFile(mainFile, []byte(`
int Foo0(void), Foo1(void), Foo2(void), Foo3(void);
int main(void) {
    return Foo0() != 0 || Foo1() != 1 || Foo2() != 2 || Foo3() != 3;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(outputDir, "archive-test.exe")
	if output, err := exec.Command("clang", mainFile, filepath.Join(outputDir, group.OutputFileName), "-o", executable).CombinedOutput(); err != nil {
		t.Fatalf("link compiled archive: %v\n%s", err, output)
	}
	if output, err := exec.Command(executable).CombinedOutput(); err != nil {
		t.Fatalf("run linked executable: %v\n%s", err, output)
	}
}

func TestCompileFailureDoesNotPublishArchiveOrLeaveObjects(t *testing.T) {
	sourceRoot := t.TempDir()
	group := CompileGroup{OutputFileName: "failed.a"}
	for i := range 8 {
		file := filepath.Join(sourceRoot, fmt.Sprintf("source%d.c", i))
		contents := fmt.Sprintf("int Foo%d(void) { return %d; }\n", i, i)
		if i == 0 {
			contents = "#error deliberately invalid source\n"
		}
		if err := os.WriteFile(file, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
		group.Files = append(group.Files, file)
	}
	outputDir := t.TempDir()
	tempRoot := t.TempDir()
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, tempRoot)
	}
	err := group.Compile(outputDir, CompileOptions{CC: "clang"})
	if err == nil || !strings.Contains(err.Error(), group.Files[0]) {
		t.Fatalf("error = %v, want the failing source path", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, group.OutputFileName)); !os.IsNotExist(err) {
		t.Fatalf("failed compilation published an archive: %v", err)
	}
	entries, err := os.ReadDir(tempRoot)
	if err != nil || len(entries) != 0 {
		t.Fatalf("temporary objects remain: %v, %v", entries, err)
	}
}

func TestCompileObjectCreationFailureCleansUp(t *testing.T) {
	tempRoot := t.TempDir()
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, tempRoot)
	}
	// Exceed the filesystem's component limit so a worker fails before Clang
	// starts. Filesystem errors must also reach the caller without publishing
	// an archive or leaving the group's temporary directory behind.
	group := CompileGroup{
		OutputFileName: "failed.a",
		Files:          []string{strings.Repeat("x", 300) + ".c"},
	}
	outputDir := t.TempDir()
	err := group.Compile(outputDir, CompileOptions{CC: "missing-clang"})
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) || pathErr.Op != "open" {
		t.Fatalf("Compile error = %v, want object creation failure", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, group.OutputFileName)); !os.IsNotExist(err) {
		t.Fatalf("failed object creation published an archive: %v", err)
	}
	entries, err := os.ReadDir(tempRoot)
	if err != nil || len(entries) != 0 {
		t.Fatalf("temporary objects remain: %v, %v", entries, err)
	}
}

func TestLibConfig_String(t *testing.T) {
	tests := []struct {
		name     string
		config   LibConfig
		expected string
	}{
		{
			name: "Normal name and version",
			config: LibConfig{
				Name:    "picolibc",
				Version: "1.0",
			},
			expected: "picolibc-1.0",
		},
		{
			name: "Empty name",
			config: LibConfig{
				Name:    "",
				Version: "2.5",
			},
			expected: "-2.5",
		},
		{
			name: "Empty version",
			config: LibConfig{
				Name:    "musl",
				Version: "",
			},
			expected: "musl-",
		},
		{
			name: "Both empty",
			config: LibConfig{
				Name:    "",
				Version: "",
			},
			expected: "-",
		},
		{
			name: "Special characters",
			config: LibConfig{
				Name:    "glibc++",
				Version: "v3.2.1",
			},
			expected: "glibc++-v3.2.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}
