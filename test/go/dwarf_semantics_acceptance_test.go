package gotest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const dwarfReturnOrderProbe = `package main

type value struct {
	n int
}

func (v *value) mutate() bool {
	v.n = 1
	return true
}

func result() (value, bool) {
	var v value
	return v, v.mutate()
}

func main() {
	v, ok := result()
	if !ok || v.n != 1 {
		panic("return value was loaded before mutation")
	}
	println("RETURN_ORDER_OK")
}
`

func TestDWARFReturnOrderSemantics(t *testing.T) {
	repoRoot := findRepoRoot(t)
	source := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(source, []byte(dwarfReturnOrderProbe), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(
		"go", "run", "./cmd/llgo", "run", "-ldflags=-w=false", source,
	)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"LLGO_ROOT="+repoRoot,
		"LLGO_BUILD_CACHE=off",
		"GOMAXPROCS=2",
		"GOMEMLIMIT=6GiB",
		"GOFLAGS=-p=1",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("LLGo DWARF return-order acceptance failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "RETURN_ORDER_OK") {
		t.Fatalf("LLGo DWARF return-order acceptance did not report success:\n%s", out)
	}
}
