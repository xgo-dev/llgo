package build

import (
	"runtime"
	"testing"
)

func TestParallelObjectEmitEnabled(t *testing.T) {
	t.Run("native host default", func(t *testing.T) {
		t.Setenv(llgoParallelObjectEmit, "")
		ctx := &context{buildConf: &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH}}
		if !parallelObjectEmitEnabled(ctx) {
			t.Fatal("expected native host build to use async object emission by default")
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
