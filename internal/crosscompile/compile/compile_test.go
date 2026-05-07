//go:build !llgo

package compile

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/goplus/llgo/xtool/nm"
)

func skipIfRootPermissionTest(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" && os.Geteuid() == 0 {
		t.Skip("permission-denial checks are not reliable when running as root")
	}
}

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

func TestCompileVerbose(t *testing.T) {
	if compileVerbose(CompileOptions{}) {
		t.Fatal("compileVerbose should be false by default")
	}
	if !compileVerbose(CompileOptions{Verbose: true}) {
		t.Fatal("compileVerbose should honor CompileOptions.Verbose")
	}
	t.Setenv("LLGO_COMPILE_VERBOSE", "1")
	if !compileVerbose(CompileOptions{}) {
		t.Fatal("compileVerbose should honor LLGO_COMPILE_VERBOSE")
	}
}

func TestCompileJobs(t *testing.T) {
	t.Setenv("LLGO_COMPILE_JOBS", "3")
	jobs, err := compileJobs()
	if err != nil {
		t.Fatalf("compileJobs: %v", err)
	}
	if jobs != 3 {
		t.Fatalf("compileJobs = %d, want 3", jobs)
	}

	t.Setenv("LLGO_COMPILE_JOBS", "0")
	if _, err := compileJobs(); err == nil {
		t.Fatal("compileJobs should reject non-positive LLGO_COMPILE_JOBS")
	}

	t.Setenv("LLGO_COMPILE_JOBS", "bad")
	if _, err := compileJobs(); err == nil {
		t.Fatal("compileJobs should reject invalid LLGO_COMPILE_JOBS")
	}

	t.Setenv("LLGO_COMPILE_JOBS", "")
	oldProcs := runtime.GOMAXPROCS(32)
	defer runtime.GOMAXPROCS(oldProcs)
	jobs, err = compileJobs()
	if err != nil {
		t.Fatalf("compileJobs default: %v", err)
	}
	if jobs != 16 {
		t.Fatalf("compileJobs should cap default jobs at 16, got %d", jobs)
	}
}

func TestCompileSkipDoesNotReadJobs(t *testing.T) {
	tmpDir := t.TempDir()
	archive := filepath.Join(tmpDir, "exists.a")
	if err := os.WriteFile(archive, []byte("archive already exists"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_COMPILE_JOBS", "bad")

	group := CompileGroup{OutputFileName: "exists.a"}
	if err := group.Compile(tmpDir, CompileOptions{CC: "clang"}); err != nil {
		t.Fatalf("Compile skipped group with invalid jobs env: %v", err)
	}
	if err := (CompileConfig{Groups: []CompileGroup{group}}).Compile(tmpDir, CompileOptions{CC: "clang"}); err != nil {
		t.Fatalf("CompileConfig skipped group with invalid jobs env: %v", err)
	}
}

func TestCompileConfigRejectsInvalidJobsWithWork(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "one.c")
	if err := os.WriteFile(src, []byte("int one(void) { return 1; }\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("LLGO_COMPILE_JOBS", "bad")
	cfg := CompileConfig{Groups: []CompileGroup{
		{OutputFileName: "libone.a", Files: []string{src}},
	}}
	if err := cfg.Compile(tmpDir, CompileOptions{CC: "clang"}); err == nil || !strings.Contains(err.Error(), "invalid LLGO_COMPILE_JOBS") {
		t.Fatalf("CompileConfig invalid jobs error = %v, want invalid LLGO_COMPILE_JOBS", err)
	}
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
		skipIfRootPermissionTest(t)

		tmpDir := filepath.Join(t.TempDir(), "test-compile")
		os.RemoveAll(tmpDir)

		err := os.Mkdir(tmpDir, 0)
		if err != nil {
			t.Error(err)
			return
		}
		defer os.RemoveAll(tmpDir)

		os.Setenv("TMPDIR", tmpDir)
		defer os.Unsetenv("TMPDIR")

		group := CompileGroup{
			OutputFileName: "nop.a",
		}
		err = group.Compile(tmpDir, CompileOptions{
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

func TestCompileConfigMultipleGroups(t *testing.T) {
	tmpDir := t.TempDir()

	writeC := func(name, symbol string) string {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("int "+symbol+"(void) { return 1; }\n"), 0644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	cfg := CompileConfig{Groups: []CompileGroup{
		{OutputFileName: "libone.a", Files: []string{writeC("one.c", "One")}},
		{OutputFileName: "libtwo.a", Files: []string{writeC("two.c", "Two")}},
	}}
	if err := cfg.Compile(tmpDir, CompileOptions{CC: "clang"}); err != nil {
		t.Fatalf("CompileConfig.Compile: %v", err)
	}

	for _, tt := range []struct {
		archive string
		symbol  string
	}{
		{"libone.a", "One"},
		{"libtwo.a", "Two"},
	} {
		archive := filepath.Join(tmpDir, tt.archive)
		items, err := nm.New("").List(archive)
		if err != nil {
			t.Fatalf("nm %s: %v", tt.archive, err)
		}
		found := false
		for _, item := range items {
			for _, sym := range item.Symbols {
				if strings.Contains(sym.Name, tt.symbol) {
					found = true
				}
			}
		}
		if !found {
			t.Fatalf("cannot find symbol %s in %s", tt.symbol, tt.archive)
		}
	}
}

func TestCompileSingleGroupParallel(t *testing.T) {
	tmpDir := t.TempDir()
	writeC := func(name, symbol string) string {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("int "+symbol+"(void) { return 1; }\n"), 0644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	t.Setenv("LLGO_COMPILE_JOBS", "2")
	group := CompileGroup{
		OutputFileName: "libsingle.a",
		Files: []string{
			writeC("one.c", "SingleOne"),
			writeC("two.c", "SingleTwo"),
		},
	}
	if err := group.Compile(tmpDir, CompileOptions{CC: "clang"}); err != nil {
		t.Fatalf("Compile single group in parallel: %v", err)
	}
	archive := filepath.Join(tmpDir, "libsingle.a")
	items, err := nm.New("").List(archive)
	if err != nil {
		t.Fatalf("nm %s: %v", archive, err)
	}
	for _, want := range []string{"SingleOne", "SingleTwo"} {
		found := false
		for _, item := range items {
			for _, sym := range item.Symbols {
				if strings.Contains(sym.Name, want) {
					found = true
				}
			}
		}
		if !found {
			t.Fatalf("cannot find symbol %s in %s", want, archive)
		}
	}
}

func TestCompileSingleGroupParallelError(t *testing.T) {
	tmpDir := t.TempDir()
	good := filepath.Join(tmpDir, "good.c")
	if err := os.WriteFile(good, []byte("int good(void) { return 1; }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(tmpDir, "bad.c")
	if err := os.WriteFile(bad, []byte("int bad(void) {\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("LLGO_COMPILE_JOBS", "2")
	group := CompileGroup{OutputFileName: "libbad.a", Files: []string{good, bad}}
	if err := group.Compile(tmpDir, CompileOptions{CC: "clang"}); err == nil {
		t.Fatal("Compile should report clang error from a bad source file")
	}
}

func TestCompileConfigMultipleGroupsParallelError(t *testing.T) {
	tmpDir := t.TempDir()
	good := filepath.Join(tmpDir, "good.c")
	if err := os.WriteFile(good, []byte("int good(void) { return 1; }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(tmpDir, "bad.c")
	if err := os.WriteFile(bad, []byte("int bad(void) {\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("LLGO_COMPILE_JOBS", "2")
	cfg := CompileConfig{Groups: []CompileGroup{
		{OutputFileName: "libgood.a", Files: []string{good}},
		{OutputFileName: "libbad.a", Files: []string{bad}},
	}}
	if err := cfg.Compile(tmpDir, CompileOptions{CC: "clang"}); err == nil {
		t.Fatal("CompileConfig should report clang error from a bad source file")
	}
}

func TestArchiveGroupsReportsArchiveErrors(t *testing.T) {
	tmpDir := t.TempDir()
	missingToolOptions := CompileOptions{CC: filepath.Join(tmpDir, "missing-clang")}
	states := []compileGroupState{
		{group: CompileGroup{OutputFileName: "libone.a"}},
		{group: CompileGroup{OutputFileName: "libtwo.a"}},
	}

	if err := archiveGroups(tmpDir, missingToolOptions, states[:1], 1); err == nil {
		t.Fatal("archiveGroups should report serial archive errors")
	}
	if err := archiveGroups(tmpDir, missingToolOptions, states, 2); err == nil {
		t.Fatal("archiveGroups should report parallel archive errors")
	}
}

func TestCompileWorkerPoolsStopAfterFirstError(t *testing.T) {
	tmpDir := t.TempDir()
	bad := filepath.Join(tmpDir, "bad.c")
	if err := os.WriteFile(bad, []byte("int bad(void) {\n"), 0644); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(tmpDir, "good.c")
	if err := os.WriteFile(good, []byte("int good(void) { return 1; }\n"), 0644); err != nil {
		t.Fatal(err)
	}

	singleGroup := compileGroupState{
		group: CompileGroup{OutputFileName: "libsingle.a", Files: []string{bad, good, good}},
		cfg:   groupCompilerConfig(CompileGroup{Files: []string{bad, good, good}}, CompileOptions{CC: "clang"}),
		tmp:   t.TempDir(),
		objs:  make([]string, 3),
	}
	if err := compileSingleGroupTasks(&singleGroup, 1, false); err == nil {
		t.Fatal("compileSingleGroupTasks should stop and report the first compile error")
	}

	multiGroups := []compileGroupState{
		{
			group: CompileGroup{OutputFileName: "libbad.a", Files: []string{bad}},
			cfg:   groupCompilerConfig(CompileGroup{Files: []string{bad}}, CompileOptions{CC: "clang"}),
			tmp:   t.TempDir(),
			objs:  make([]string, 1),
		},
		{
			group: CompileGroup{OutputFileName: "libgood.a", Files: []string{good, good}},
			cfg:   groupCompilerConfig(CompileGroup{Files: []string{good, good}}, CompileOptions{CC: "clang"}),
			tmp:   t.TempDir(),
			objs:  make([]string, 2),
		},
	}
	tasks := []compileTask{
		{group: 0, file: 0},
		{group: 1, file: 0},
		{group: 1, file: 1},
	}
	if err := compileTasks(multiGroups, tasks, 1, false); err == nil {
		t.Fatal("compileTasks should stop and report the first compile error")
	}
}

func TestSingleTaskGroup(t *testing.T) {
	if _, ok := singleTaskGroup(nil); ok {
		t.Fatal("empty task list should not be a single group")
	}
	if group, ok := singleTaskGroup([]compileTask{{group: 2}, {group: 2}}); !ok || group != 2 {
		t.Fatalf("singleTaskGroup same group = %d, %v; want 2, true", group, ok)
	}
	if _, ok := singleTaskGroup([]compileTask{{group: 1}, {group: 2}}); ok {
		t.Fatal("mixed task groups should not report a single group")
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
