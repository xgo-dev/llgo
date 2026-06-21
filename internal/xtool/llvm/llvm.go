package llvm

import "runtime"

func GetTargetTriple(goos, goarch string) string {
	var llvmarch string
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	if goos == "" {
		goos = runtime.GOOS
	}
	switch goarch {
	case "386":
		llvmarch = "i386"
	case "amd64":
		llvmarch = "x86_64"
	case "arm64":
		llvmarch = "aarch64"
	case "arm":
		// Keep the default in sync with ssa.Target.Spec when GOARM is not
		// explicitly modeled by this helper.
		llvmarch = "armv7"
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
			llvmos = "macosx"
		}
		llvmvendor = "apple"
	case "wasip1":
		llvmos = "wasip1"
	}
	// Target triples (which actually have four components, but are called
	// triples for historical reasons) have the form:
	//   arch-vendor-os-environment
	triple := llvmarch + "-" + llvmvendor + "-" + llvmos
	if llvmos == "windows" {
		triple += "-gnu"
	} else if goarch == "arm" {
		triple += "-gnueabihf"
	}
	return triple
}
