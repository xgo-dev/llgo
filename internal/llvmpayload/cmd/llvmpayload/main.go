// Command llvmpayload prints the checked-in LLVM payload contract for release
// scripts that cannot import Go constants directly.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xgo-dev/llgo/internal/llvmpayload"
)

func main() {
	if run(os.Args[1:], os.Stdout, os.Stderr) != 0 {
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("llvmpayload", flag.ContinueOnError)
	flags.SetOutput(stderr)
	major := flags.Int("major", 0, "LLVM major (defaults to the release payload)")
	if err := flags.Parse(args); err != nil {
		return 1
	}

	var (
		manifest llvmpayload.Manifest
		err      error
	)
	if *major == 0 {
		manifest, err = llvmpayload.Default()
	} else {
		manifest, err = llvmpayload.ForMajor(*major)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	fmt.Fprintf(stdout, "LLGO_LLVM_MAJOR=%s\n", fmt.Sprint(manifest.LLVMMajor()))
	fmt.Fprintf(stdout, "ESP_CLANG_LLVM_MAJOR=%s\n", fmt.Sprint(manifest.LLVMMajor()))
	fmt.Fprintf(stdout, "ESP_CLANG_VERSION=%s\n", manifest.Version())
	fmt.Fprintf(stdout, "ESP_CLANG_BASE_URL=%s\n", manifest.BaseURL())
	for _, host := range []struct {
		name, goos, goarch string
	}{
		{name: "DARWIN_AMD64", goos: "darwin", goarch: "amd64"},
		{name: "DARWIN_ARM64", goos: "darwin", goarch: "arm64"},
		{name: "LINUX_AMD64", goos: "linux", goarch: "amd64"},
		{name: "LINUX_ARM64", goos: "linux", goarch: "arm64"},
	} {
		platform, ok := llvmpayload.PlatformSuffix(host.goos, host.goarch)
		if !ok {
			panic("missing platform mapping for " + host.goos + "/" + host.goarch)
		}
		artifact, err := manifest.Artifact(platform)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "ESP_CLANG_SHA256_%s=%s\n", strings.ToUpper(host.name), artifact.SHA256)
	}
	return 0
}
