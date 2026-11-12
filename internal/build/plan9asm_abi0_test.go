package build

import (
	stdcontext "context"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/packages"
	llplan9asm "github.com/xgo-dev/llgo/internal/plan9asm"
)

func abi0FixtureCommand(t *testing.T, timeout time.Duration, name string, args ...string) *exec.Cmd {
	t.Helper()
	ctx, cancel := stdcontext.WithTimeout(t.Context(), timeout)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = 5 * time.Second
	return cmd
}

// Run on every host: the fixture must be accepted by the actual Go assembler
// and lower to an LLVM object. This is not a substitute for the native caller.
func TestPlan9AsmABI0Objects(t *testing.T) {
	dir, err := filepath.Abs("testdata/plan9asm-abi0")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(dir, "main.go"), nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	sizes := types.SizesFor("gc", "amd64")
	config := types.Config{Sizes: sizes}
	typed, err := config.Check("example.com/abi0probe", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	pkg := &packages.Package{
		PkgPath: typed.Path(), Dir: dir, Types: typed, TypesSizes: sizes,
		Fset: fset, Syntax: []*ast.File{file},
	}
	clangVersion, err := abi0FixtureCommand(t, 10*time.Second, "clang", "--version").CombinedOutput()
	if err != nil || !strings.Contains(string(clangVersion), "clang version 22.") {
		t.Fatalf("LLVM 22 clang is required: %v\n%s", err, clangVersion)
	}
	for _, target := range []struct{ goos, triple string }{
		{"linux", "x86_64-unknown-linux-gnu"},
		{"darwin", "x86_64-apple-darwin"},
	} {
		t.Run(target.goos, func(t *testing.T) {
			outDir := t.TempDir()
			cmd := abi0FixtureCommand(t, 2*time.Minute, "go", "build", "-o", filepath.Join(outDir, "gc-probe"), ".")
			cmd.Dir = dir
			cmd.Env = withEnv(os.Environ(), "GOOS="+target.goos, "GOARCH=amd64", "GOAMD64=v1", "CGO_ENABLED=0", "GOTOOLCHAIN=local")
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("original Go assembly: %v\n%s", err, output)
			}
			translated, err := llplan9asm.TranslateFileForPkgWithOptions(pkg,
				filepath.Join(dir, "wrappers_amd64.s"), target.goos, "amd64", nil,
				llplan9asm.TranslateOptions{GOROOT: runtime.GOROOT(), GOAMD64: "v1"})
			if err != nil {
				t.Fatal(err)
			}
			if len(translated.Functions) != 2 {
				t.Fatalf("missing ABI0/VMOVQ functions: %+v", translated.Functions)
			}
			llPath := filepath.Join(outDir, "wrappers.ll")
			if err := os.WriteFile(llPath, []byte(translated.LLVMIR), 0o600); err != nil {
				t.Fatal(err)
			}
			cmd = abi0FixtureCommand(t, time.Minute, "clang", "--target="+target.triple, "-Wno-override-module", "-c", llPath, "-o", filepath.Join(outDir, "wrappers.o"))
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("LLVM 22 assembly object: %v\n%s", err, output)
			}
		})
	}
}

// Unlike the adapter's IR checks, this builds Go callees with LLGo, links the
// translated assembly and executes the caller. It covers the reduced ABI0
// contract from plan9asm#44 / llgo#2691, not the entire modernc libc package.
func TestPlan9AsmABI0Integration(t *testing.T) {
	if runtime.GOARCH != "amd64" || runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("requires native linux/amd64 or darwin/amd64; cross-target IR is checked separately")
	}
	dir, err := filepath.Abs("testdata/plan9asm-abi0")
	if err != nil {
		t.Fatal(err)
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(t.TempDir(), "abi0-probe")
	if _, err := Build(Invocation{Dir: dir, Args: []string{"."}, Config: conf}); err != nil {
		t.Fatal(err)
	}
	out, err := abi0FixtureCommand(t, 10*time.Second, conf.OutFile).CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "ok" {
		t.Fatalf("ABI0 caller: %v\n%s", err, out)
	}
}
