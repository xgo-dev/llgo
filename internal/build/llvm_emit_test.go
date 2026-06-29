package build

import (
	"runtime"
	"testing"

	"github.com/goplus/llgo/internal/lto"
)

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

	t.Run("full LTO", func(t *testing.T) {
		conf := &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH, LTO: lto.Full}
		if !useInMemoryNativeCodegenConf(conf) {
			t.Fatal("expected native full LTO build to use in-memory bitcode emission")
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
