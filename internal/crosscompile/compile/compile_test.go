//go:build !llgo

package compile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
