//go:build !llgo
// +build !llgo

package llvm

import (
	"os/exec"
	"strings"
	"testing"
)

func TestGetTargetTriple(t *testing.T) {
	// Get the list of supported architectures from clang
	cmd := exec.Command("clang", "--print-targets")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run clang --print-targets: %v", err)
	}

	// Parse the output to get the list of supported architectures
	supportedArchs := make(map[string]bool)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "Registered Targets:") {
			// Extract the architecture from the line
			parts := strings.SplitN(line, " - ", 2)
			if len(parts) > 0 {
				arch := strings.TrimSpace(parts[0])
				supportedArchs[arch] = true
			}
		}
	}

	if len(supportedArchs) == 0 {
		t.Fatal("No supported architectures found from clang --print-targets")
	}

	t.Logf("Found %d supported architectures from clang", len(supportedArchs))

	// Map our architecture names to clang's architecture names
	clangArchMap := map[string][]string{
		"x86_64":  {"x86-64", "x86_64"},
		"i386":    {"x86", "i386"},
		"aarch64": {"aarch64", "arm64"},
		"arm64":   {"arm64", "aarch64"},
		"armv7":   {"arm", "thumb"},
		"wasm32":  {"wasm32"},
	}

	// Define a function to check if the architecture is supported by clang
	isArchSupported := func(archPart string) (bool, string) {
		if mappedArchs, ok := clangArchMap[archPart]; ok {
			for _, mappedArch := range mappedArchs {
				if supportedArchs[mappedArch] {
					return true, mappedArch
				}
			}
		} else if supportedArchs[archPart] {
			// Direct match
			return true, archPart
		}
		return false, ""
	}

	// Define a function to verify OS name
	isOSValid := func(os, goos string) bool {
		validOSMap := map[string][]string{
			"linux":   {"linux", "linux-gnu"},
			"darwin":  {"macosx", "darwin"},
			"windows": {"windows", "win32"},
			"wasip1":  {"wasip1", "wasi"},
			"js":      {"js", "javascript"},
		}

		if validVariants, ok := validOSMap[goos]; ok {
			for _, validVariant := range validVariants {
				if strings.HasPrefix(os, validVariant) {
					return true
				}
			}
		}
		return false
	}

	// Define a function to check if vendor is valid
	isVendorValid := func(vendor string) bool {
		validVendors := map[string]bool{
			"unknown": true,
			"apple":   true,
			"pc":      true,
			"ibm":     true,
		}
		return validVendors[vendor]
	}

	// Define the test function
	checkTriple := func(t *testing.T, testName, goos, goarch, expected string) {
		t.Helper()
		got := GetTargetTriple(goos, goarch)

		// Check if the generated triple matches the expected value
		if got != expected {
			t.Errorf("getTargetTriple(%q, %q) = %q, want %q",
				goos, goarch, got, expected)
		}

		// Extract the architecture part from the triple (first component)
		parts := strings.Split(got, "-")
		if len(parts) < 3 {
			t.Errorf("Invalid target triple format: %s, should have at least 3 components", got)
			return
		}

		archPart := parts[0]
		vendor := parts[1]
		os := parts[2]

		// Check if the architecture is supported by clang
		supported, mappedArch := isArchSupported(archPart)
		if supported {
			t.Logf("Architecture %s (mapped to %s) is supported by clang", archPart, mappedArch)
		} else {
			t.Logf("WARNING: Architecture %s from triple %q for %s/%s not found in clang's supported architectures",
				archPart, got, goos, goarch)
		}

		// Verify vendor
		if !isVendorValid(vendor) {
			t.Errorf("Invalid vendor in triple: %s", vendor)
		}

		// Verify OS
		if !isOSValid(os, goos) {
			t.Errorf("OS in triple %q doesn't match expected OS %q", os, goos)
		}
	}

	// Run tests for different OS/arch combinations
	checkTriple(t, "wasip1/wasm", "wasip1", "wasm", "wasm32-unknown-wasip1")
	checkTriple(t, "linux/amd64", "linux", "amd64", "x86_64-unknown-linux")
	checkTriple(t, "linux/386", "linux", "386", "i386-unknown-linux")
	checkTriple(t, "linux/arm64", "linux", "arm64", "aarch64-unknown-linux")
	checkTriple(t, "linux/arm", "linux", "arm", "armv7-unknown-linux-gnueabihf")
	checkTriple(t, "darwin/amd64", "darwin", "amd64", "x86_64-apple-macosx")
	checkTriple(t, "darwin/arm64", "darwin", "arm64", "arm64-apple-macosx")
	checkTriple(t, "windows/amd64", "windows", "amd64", "x86_64-unknown-windows-gnu")
	checkTriple(t, "windows/386", "windows", "386", "i386-unknown-windows-gnu")
	checkTriple(t, "js/wasm", "js", "wasm", "wasm32-unknown-js")
}
