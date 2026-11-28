package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestPlan9AsmARM64RuntimeMemmove(t *testing.T) {
	if runtime.GOARCH != "arm64" || runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native ARM64 execution; typed outgoing ABI is checked separately")
	}
	setPCLNIntegrationEnv(t)
	t.Setenv("LLGO_PLAN9ASM_PKGS", "all")
	directory, err := filepath.Abs("testdata/plan9asm-memmove-arm64")
	if err != nil {
		t.Fatal(err)
	}
	for _, compiler := range []string{"go", "llgo"} {
		t.Run(compiler, func(t *testing.T) {
			binary := filepath.Join(t.TempDir(), "memmove")
			if compiler == "go" {
				command := abi0FixtureCommand(t, time.Minute, filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-o", binary, ".")
				command.Dir = directory
				command.Env = withEnv(os.Environ(), "CGO_ENABLED=0", "GOTOOLCHAIN=local")
				if output, err := command.CombinedOutput(); err != nil {
					t.Fatalf("stock Go: %v\n%s", err, output)
				}
			} else {
				conf := NewDefaultConf(ModeBuild)
				conf.OutFile = binary
				if _, err := Build(Invocation{Dir: directory, Args: []string{"."}, Config: conf}); err != nil {
					t.Fatal(err)
				}
			}
			output, err := abi0FixtureCommand(t, time.Minute, binary).CombinedOutput()
			if err != nil || strings.TrimSpace(string(output)) != "ARM64 ABI0 memmove verified" {
				t.Fatalf("execute: %v\n%s", err, output)
			}
		})
	}
}
