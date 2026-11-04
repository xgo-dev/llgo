//go:build (darwin || linux) && (amd64 || arm64)

package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPlan9AsmNativeComponentOriginalGo(t *testing.T) {
	for _, slots := range []int{4, 2000} {
		t.Run(strconv.Itoa(slots), func(t *testing.T) {
			for _, layout := range []struct {
				name  string
				split bool
			}{{"inlineDATA", false}, {"splitDATA", true}} {
				t.Run(layout.name, func(t *testing.T) {
					files := nativeComponentTableFiles(t, slots)
					nativeComponentDataLayout(t, files, layout.split)
					directory := nativeAddressFixtureWrite(t, files)
					binary := filepath.Join(t.TempDir(), "original")
					command := abi0FixtureCommand(t, 2*time.Minute, filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-p=1", "-mod=readonly", "-o", binary, ".")
					command.Dir = directory
					command.Env = withEnv(os.Environ(), "GOROOT="+runtime.GOROOT(), "GOTOOLCHAIN=local", "GOENV=off", "GOWORK=off", "GOFLAGS=", "CGO_ENABLED=0", "GOPROXY=off")
					if output, err := command.CombinedOutput(); err != nil {
						t.Fatalf("complete original Go component package/assembly/link: %v\n%s", err, output)
					}
					output, err := abi0FixtureCommand(t, 10*time.Second, binary).CombinedOutput()
					if err != nil || strings.TrimSpace(string(output)) != "native component addresses verified" {
						t.Fatalf("original Go component address observation: %v\n%s", err, output)
					}
				})
			}
		})
	}
}

func TestPlan9AsmNativeComponentLinkedGoObservation(t *testing.T) {
	for _, slots := range []int{4, 2000} {
		t.Run(strconv.Itoa(slots), func(t *testing.T) {
			for _, layout := range []struct {
				name  string
				split bool
			}{{"inlineDATA", false}, {"splitDATA", true}} {
				t.Run(layout.name, func(t *testing.T) {
					files := nativeComponentTableFiles(t, slots)
					nativeComponentDataLayout(t, files, layout.split)
					directory := nativeAddressFixtureWrite(t, files)
					conf := NewDefaultConf(ModeBuild)
					conf.OutFile = filepath.Join(t.TempDir(), "translated")
					if _, err := Build(Invocation{Dir: directory, Args: []string{"."}, Config: conf}); err != nil {
						t.Fatalf("LLGo component assembly objects/link: %v", err)
					}
					output, err := abi0FixtureCommand(t, 10*time.Second, conf.OutFile).CombinedOutput()
					if err != nil || strings.TrimSpace(string(output)) != "native component addresses verified" {
						t.Fatalf("translated component address observation: %v\n%s", err, output)
					}
				})
			}
		})
	}
}
