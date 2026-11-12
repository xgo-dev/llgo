//go:build (darwin || linux) && (amd64 || arm64)

package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Keep these independent of a third-party module and of native assembly
// translation: each entry is compiled from C by the selected toolchain.
func TestPackedRuntimeCallsOriginalGo(t *testing.T) {
	testPackedRuntimeCalls(t, false)
}

func TestPackedRuntimeCallsLLGo(t *testing.T) {
	testPackedRuntimeCalls(t, true)
}

func testPackedRuntimeCalls(t *testing.T, translated bool) {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	t.Setenv("CGO_ENABLED", "1")
	// The test binary itself links LLVM; the independent C probes do not.
	t.Setenv("CGO_CPPFLAGS", "")
	t.Setenv("CGO_LDFLAGS", "")
	for _, fixture := range []struct {
		name  string
		modes []string
	}{
		{name: "packed-cgocall", modes: []string{"packed"}},
		{name: "packed-crosscall2", modes: []string{"same-thread", "nested", "foreign-thread"}},
	} {
		if fixture.name == "packed-crosscall2" && runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
			fixture.modes = append(fixture.modes, "stack-aligned", "stack-eight-aligned")
		}
		t.Run(fixture.name, func(t *testing.T) {
			directory, err := filepath.Abs(filepath.Join("testdata", fixture.name))
			if err != nil {
				t.Fatal(err)
			}
			binary := filepath.Join(t.TempDir(), "probe")
			if translated {
				conf := NewDefaultConf(ModeBuild)
				conf.OutFile = binary
				if _, err := Build(Invocation{Dir: directory, Args: []string{"."}, Config: conf}); err != nil {
					t.Fatalf("LLGo complete C/Go build and link: %v", err)
				}
			} else {
				command := abi0FixtureCommand(t, 2*time.Minute,
					filepath.Join(runtime.GOROOT(), "bin", "go"),
					"build", "-p=1", "-mod=readonly", "-o", binary, ".")
				command.Dir = directory
				command.Env = withEnv(os.Environ(), "GOROOT="+runtime.GOROOT(),
					"GOTOOLCHAIN=local", "GOENV=off", "GOWORK=off", "GOFLAGS=", "CGO_ENABLED=1", "GOPROXY=off")
				if output, err := command.CombinedOutput(); err != nil {
					t.Fatalf("original Go complete C/Go build and link: %v\n%s", err, output)
				}
			}
			for _, mode := range fixture.modes {
				t.Run(mode, func(t *testing.T) {
					output, err := abi0FixtureCommand(t, 30*time.Second, binary, mode).CombinedOutput()
					want := mode + " verified 128"
					if err != nil || strings.TrimSpace(string(output)) != want {
						t.Fatalf("packed entry execution: %v\noutput: %s\nwant: %s", err, output, want)
					}
				})
			}
		})
	}
}
