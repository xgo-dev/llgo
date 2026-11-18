package goroot

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBaselineCgoEnvironmentDoesNotChangeLLGoTarget(t *testing.T) {
	llgoEnv := []string{
		"CC=clang --target=i686-pc-windows-msvc", "CXX=clang++ --target=i686-pc-windows-msvc",
		"CGO_CPPFLAGS=-I/host-llvm", "CGO_LDFLAGS=-lLLVM", "GOARCH=386",
		`LLGO_GOROOT_BASELINE_CC="C:/tool chain/i686-w64-mingw32-clang.exe" -fuse-ld=lld`,
		`LLGO_GOROOT_BASELINE_CXX="C:/tool chain/i686-w64-mingw32-clang++.exe" -fuse-ld=lld`,
	}
	original := append([]string{}, llgoEnv...)
	baseline := baselineCgoEnv(llgoEnv)
	if !reflect.DeepEqual(llgoEnv, original) {
		t.Fatal("reference compiler selection changed the LLGo environment")
	}
	for _, key := range []string{"CC", "CXX"} {
		if got := caseEnvValue(baseline, key); got != caseEnvValue(llgoEnv, "LLGO_GOROOT_BASELINE_"+key) {
			t.Fatalf("reference %s = %q", key, got)
		}
	}
	for _, key := range []string{"CGO_CFLAGS", "CGO_CPPFLAGS", "CGO_CXXFLAGS", "CGO_FFLAGS", "CGO_LDFLAGS"} {
		if caseEnvValue(baseline, key) != "" {
			t.Fatalf("reference program inherited host %s", key)
		}
	}
	if caseEnvValue(baseline, "GOARCH") != "386" {
		t.Fatal("reference architecture changed")
	}
	plain := []string{"CC=custom-cc", "CGO_CFLAGS=-DCASE=1"}
	if got := baselineCgoEnv(plain); !reflect.DeepEqual(got, plain) {
		t.Fatal("changed an environment without a reference compiler override")
	}
}

func TestPrintConstantNormalizationPreservesAssertions(t *testing.T) {
	source := `package main
import "unknown.invalid/package"
const C32 = (-1) & ((1 << 32) - 1)
func main() {
 if C32 != 0xFFFFFFFF { println("bad", C32); panic("fail") }
 print(-(1<<63), "\n")
 println(1<<64-1)
}
`
	name := filepath.Join(t.TempDir(), "in.go")
	if err := os.WriteFile(name, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if n, err := normalizePrintConstants(name, "386"); err != nil || n != 3 {
		t.Fatalf("normalize = %d, %v; want 3, nil", n, err)
	}
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "\n") != strings.Count(source, "\n") || !strings.Contains(string(data), `if C32 != 0xFFFFFFFF { println("bad", int64(C32)); panic("fail") }`) {
		t.Fatalf("normalization changed assertion logic or line numbers:\n%s", data)
	}
	// Import loading is intentionally omitted by normalization. Strip the
	// unresolved import only for this independent 386 type-check assertion.
	data = []byte(strings.Replace(string(data), "import \"unknown.invalid/package\"\n", "", 1))
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, data, 0)
	if err != nil {
		t.Fatal(err)
	}
	config := types.Config{Sizes: types.SizesFor("gc", "386")}
	if _, err := config.Check("main", fset, []*ast.File{file}, nil); err != nil {
		t.Fatalf("normalized program is not valid on 386: %v", err)
	}
	if n, err := normalizePrintConstants(name, "386"); err != nil || n != 0 {
		t.Fatalf("normalization is not idempotent: %d, %v", n, err)
	}
}

func TestPrintConstantNormalizationLeavesOtherCodeIntact(t *testing.T) {
	for _, source := range []string{
		`package main; func main() { println(42, uint64(1<<63)) }`,
		`package main; func println(uint64) {}; func main() { println(1<<63) }`,
		`package main; func main() { int64 := 1; _ = int64; println(1<<40) }`,
		`package main; func main() { println(1<<100) }`,
	} {
		name := filepath.Join(t.TempDir(), "in.go")
		if err := os.WriteFile(name, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		if n, err := normalizePrintConstants(name, "386"); err != nil || n != 0 {
			t.Fatalf("unexpected normalization %d, %v for %s", n, err, source)
		}
		data, _ := os.ReadFile(name)
		if string(data) != source {
			t.Fatalf("changed source: %s", data)
		}
	}
}
