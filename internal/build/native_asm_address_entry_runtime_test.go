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

func TestPlan9AsmAddressOnlyNativeEntryOriginalGo(t *testing.T) {
	directory := nativeAddressFixtureWrite(t, nativeAddressFixtureFiles(t))
	binary := filepath.Join(t.TempDir(), "original")
	command := abi0FixtureCommand(t, 2*time.Minute, filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-p=1", "-mod=readonly", "-o", binary, ".")
	command.Dir = directory
	command.Env = withEnv(os.Environ(), "GOROOT="+runtime.GOROOT(), "GOTOOLCHAIN=local", "GOENV=off", "GOWORK=off", "GOFLAGS=", "CGO_ENABLED=0", "GOPROXY=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("complete original Go package/assembly/link: %v\n%s", err, output)
	}
	output, err := abi0FixtureCommand(t, 10*time.Second, binary).CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "native address relocation verified" {
		t.Fatalf("original Go code address observation: %v\n%s", err, output)
	}
}

// The program only observes the linked code address. It never casts it to a
// function, calls a guessed signature, or claims a C/native entry convention.
func TestPlan9AsmAddressOnlyNativeEntryLinkedGoObservation(t *testing.T) {
	directory := nativeAddressFixtureWrite(t, nativeAddressFixtureFiles(t))
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(t.TempDir(), "translated")
	if _, err := Build(Invocation{Dir: directory, Args: []string{"."}, Config: conf}); err != nil {
		t.Fatalf("LLGo address-only assembly object/link: %v", err)
	}
	output, err := abi0FixtureCommand(t, 10*time.Second, conf.OutFile).CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "native address relocation verified" {
		t.Fatalf("translated code address observation: %v\n%s", err, output)
	}
}
