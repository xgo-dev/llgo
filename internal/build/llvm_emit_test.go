package build

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goplus/llgo/internal/crosscompile"
)

func TestParallelObjectEmitEnabled(t *testing.T) {
	t.Run("native host default", func(t *testing.T) {
		t.Setenv(llgoParallelObjectEmit, "")
		ctx := &context{buildConf: &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH}}
		if !parallelObjectEmitEnabled(ctx) {
			t.Fatal("expected native host build to use async object emission by default")
		}
	})

	t.Run("external target default", func(t *testing.T) {
		t.Setenv(llgoParallelObjectEmit, "")
		ctx := &context{buildConf: &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH, Target: "esp32"}}
		if !parallelObjectEmitEnabled(ctx) {
			t.Fatal("expected external clang target build to use async object emission by default")
		}
	})

	t.Run("opt out", func(t *testing.T) {
		t.Setenv(llgoParallelObjectEmit, "0")
		ctx := &context{buildConf: &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH}}
		if parallelObjectEmitEnabled(ctx) {
			t.Fatal("expected LLGO_PARALLEL_OBJECT_EMIT=0 to disable async object emission")
		}
	})

	t.Run("gen ll", func(t *testing.T) {
		t.Setenv(llgoParallelObjectEmit, "")
		ctx := &context{buildConf: &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH, GenLL: true}}
		if parallelObjectEmitEnabled(ctx) {
			t.Fatal("expected GenLL builds to keep synchronous object emission")
		}
	})

	t.Run("print commands", func(t *testing.T) {
		t.Setenv(llgoParallelObjectEmit, "")
		ctx := &context{buildConf: &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH, PrintCommands: true}}
		if parallelObjectEmitEnabled(ctx) {
			t.Fatal("expected command tracing to keep synchronous object emission")
		}
	})
}

func TestExportObjectWithClangUsesTargetCompilerForExternalAsyncBuild(t *testing.T) {
	t.Setenv(llgoParallelObjectEmit, "")
	tmp := t.TempDir()
	stamp := filepath.Join(tmp, "fake-cc.args")
	stdin := filepath.Join(tmp, "fake-cc.stdin")
	cc := filepath.Join(tmp, "fake-cc")
	if err := os.WriteFile(cc, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$LLGO_FAKE_CC_STAMP\"\ncat > \"$LLGO_FAKE_CC_STDIN\"\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_FAKE_CC_STAMP", stamp)
	t.Setenv("LLGO_FAKE_CC_STDIN", stdin)

	ctx := &context{
		buildConf: &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH, Target: "esp32"},
		crossCompile: crosscompile.Export{
			CC:      cc,
			CCFLAGS: []string{"-target", "xtensa"},
		},
	}
	obj, err := exportObjectWithClang(ctx, "testpkg", filepath.Join(tmp, "testpkg.a"), "target triple = \"xtensa\"\n")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(obj)
	if _, err := os.Stat(stamp); err != nil {
		t.Fatalf("expected target compiler to be invoked: %v", err)
	}
	if got, err := os.ReadFile(stdin); err != nil {
		t.Fatalf("expected LLVM IR to be sent on stdin: %v", err)
	} else if string(got) != "target triple = \"xtensa\"\n" {
		t.Fatalf("unexpected stdin: %q", got)
	}
}

func TestUseInMemoryNativeCodegenConf(t *testing.T) {
	t.Run("native host", func(t *testing.T) {
		conf := &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH}
		if !useInMemoryNativeCodegenConf(conf) {
			t.Fatal("expected native host build to use in-memory native codegen")
		}
	})

	t.Run("embedded target", func(t *testing.T) {
		conf := &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH, Target: "rp2040"}
		if useInMemoryNativeCodegenConf(conf) {
			t.Fatal("expected embedded target build to keep using clang")
		}
	})

	t.Run("cross compile host mismatch", func(t *testing.T) {
		goos := runtime.GOOS
		goarch := runtime.GOARCH
		if goos == "linux" {
			goos = "darwin"
		} else {
			goos = "linux"
		}
		if goarch == "amd64" {
			goarch = "arm64"
		} else {
			goarch = "amd64"
		}
		conf := &Config{Goos: goos, Goarch: goarch}
		if useInMemoryNativeCodegenConf(conf) {
			t.Fatal("expected host mismatch to keep using clang")
		}
	})

	t.Run("wasm", func(t *testing.T) {
		conf := &Config{Goos: "wasip1", Goarch: "wasm"}
		if useInMemoryNativeCodegenConf(conf) {
			t.Fatal("expected wasm target to keep using clang")
		}
	})
}
