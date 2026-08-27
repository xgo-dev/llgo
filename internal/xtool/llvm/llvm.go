package llvm

import (
	"runtime"

	archcfg "github.com/xgo-dev/llgo/internal/goarch"
)

func GetTargetTriple(goos, goarch string) string {
	return GetTargetTripleWithGOARM(goos, goarch, "")
}

// GetTargetTripleWithGOARM returns the LLVM target triple for a Go target.
// goarm selects the ARM version and floating-point ABI for GOARCH=arm.
func GetTargetTripleWithGOARM(goos, goarch, goarm string) string {
	var llvmarch string
	var armConfig archcfg.ARM
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	if goos == "" {
		goos = runtime.GOOS
	}
	switch goarch {
	case "386":
		if goos == "windows" {
			// LLVM's 32-bit MSVC target spelling uses i686.
			llvmarch = "i686"
		} else {
			llvmarch = "i386"
		}
	case "amd64":
		llvmarch = "x86_64"
	case "arm64":
		llvmarch = "aarch64"
	case "arm":
		armConfig, _ = archcfg.ParseARM(goarm)
		switch armConfig.Version {
		case "5":
			llvmarch = "armv5"
		case "6":
			llvmarch = "armv6"
		default:
			llvmarch = "armv7"
		}
	case "wasm":
		llvmarch = "wasm32"
	default:
		llvmarch = goarch
	}
	llvmvendor := "unknown"
	llvmos := goos
	switch goos {
	case "darwin":
		// Use macosx* instead of darwin, otherwise darwin/arm64 will refer
		// to iOS!
		llvmos = "macosx"
		if llvmarch == "aarch64" {
			// Looks like Apple prefers to call this architecture ARM64
			// instead of AArch64.
			llvmarch = "arm64"
		}
		llvmvendor = "apple"
	case "wasip1":
		llvmos = "wasip1"
	case "windows":
		// GOOS=windows defaults to the native Microsoft ABI. MinGW is a
		// separate target toolchain and must not be inferred from the host
		// shell.
		llvmvendor = "pc"
	}
	// Target triples (which actually have four components, but are called
	// triples for historical reasons) have the form:
	//   arch-vendor-os-environment
	triple := llvmarch + "-" + llvmvendor + "-" + llvmos
	if llvmos == "windows" {
		triple += "-msvc"
	} else if goarch == "arm" {
		triple += "-gnueabi"
		if !armConfig.SoftFloat {
			triple += "hf"
		}
	} else if llvmos == "linux" {
		// Keep the GNU environment explicit. Recent Clang versions use it to
		// map an unknown-vendor target to Debian's vendor-less GCC install
		// triple (for example, aarch64-linux-gnu).
		triple += "-gnu"
	}
	return triple
}
