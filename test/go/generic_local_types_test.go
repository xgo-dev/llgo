package gotest

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func genericLocalRuntimeType[T any]() reflect.Type {
	type local struct {
		value T
	}
	return reflect.TypeOf(local{})
}

func TestGenericLocalRuntimeTypesIncludeOuterArgs(t *testing.T) {
	intType := genericLocalRuntimeType[int]()
	stringType := genericLocalRuntimeType[string]()
	sameIntType := genericLocalRuntimeType[int]()
	if intType != sameIntType {
		t.Fatalf("same local generic runtime type has different identities: %v != %v", intType, sameIntType)
	}
	if intType == stringType {
		t.Fatalf("local generic runtime types are identical: %v", intType)
	}
	intName := intType.String()
	stringName := stringType.String()
	if !strings.Contains(intName, "int") {
		t.Fatalf("local generic int type name does not include type argument: %q", intName)
	}
	if !strings.Contains(stringName, "string") {
		t.Fatalf("local generic string type name does not include type argument: %q", stringName)
	}
}

type genericLocalIntish interface{ ~int }

func genericNestedLocalRuntimeTypes[A genericLocalIntish]() (reflect.Type, reflect.Type) {
	type Int int
	type T[B genericLocalIntish] struct{}
	type U[_ any] int
	return reflect.TypeOf(T[int]{}), reflect.TypeOf(T[U[int]]{})
}

func TestGenericNestedLocalRuntimeTypeNames(t *testing.T) {
	direct, nested := genericNestedLocalRuntimeTypes[int]()
	if got, want := direct.String(), "gotest.T[int;int]"; got != want {
		t.Fatalf("direct local generic type name = %q, want %q", got, want)
	}
	nestedName := nested.String()
	const nestedPrefix = "gotest.T[int;github.com/goplus/llgo/test/go.U[int;int]·"
	if !strings.HasPrefix(nestedName, nestedPrefix) || !strings.HasSuffix(nestedName, "]") {
		t.Fatalf("nested local generic type name = %q, want %q plus numeric suffix", nestedName, nestedPrefix)
	}
	ordinal := strings.TrimSuffix(strings.TrimPrefix(nestedName, nestedPrefix), "]")
	if ordinal == "" {
		t.Fatalf("nested local generic type name = %q, want numeric local type suffix", nestedName)
	}
	for _, r := range ordinal {
		if r < '0' || r > '9' {
			t.Fatalf("nested local generic type name = %q, want numeric local type suffix", nestedName)
		}
	}
}

const genericNestedLocalCommandLineProbe = `package main

import (
	"fmt"
	"reflect"
)

type intish interface{ ~int }

func nested[A intish]() (reflect.Type, reflect.Type) {
	type Int int
	type T[B intish] struct{}
	type U[_ any] int
	return reflect.TypeOf(T[int]{}), reflect.TypeOf(T[U[int]]{})
}

func main() {
	direct, nested := nested[int]()
	fmt.Println(direct)
	fmt.Println(nested)
}
`

func TestGenericNestedLocalRuntimeTypeNamesForCommandLineMain(t *testing.T) {
	dir, err := os.MkdirTemp("", "llgo-generic-local-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte(genericNestedLocalCommandLineProbe), 0644); err != nil {
		t.Fatal(err)
	}
	repoRoot := genericLocalRepoRoot(t)
	goOut := runGenericLocalProbe(t, repoRoot, "go", "run", file)
	const want = "main.T[int;int]\nmain.T[int;main.U[int;int]·3]\n"
	if goOut != want {
		t.Fatalf("go probe output = %q, want %q", goOut, want)
	}
	t.Setenv("LLGO_ROOT", repoRoot)
	llgoOut := runGenericLocalProbe(t, repoRoot, "go", "run", "./cmd/llgo", "run", file)
	if llgoOut != goOut {
		t.Fatalf("llgo probe output = %q, want go output %q", llgoOut, goOut)
	}
}

func runGenericLocalProbe(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, stderr.String())
	}
	return string(out)
}

func genericLocalRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}
