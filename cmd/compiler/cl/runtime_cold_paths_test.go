package cl_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/build"
	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
)

func TestRuntimeColdPathsCodegen(t *testing.T) {
	llvmgen.Initialize(llvmgen.InitAll)
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	conf := build.NewDefaultConf(build.ModeGen)
	conf.Goos, conf.Goarch = "darwin", "arm64"
	pkgs, err := build.Build(build.Invocation{Args: []string{llvmgen.PkgRuntime}, Config: conf})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].LPkg == nil {
		t.Fatal("missing compiled runtime package")
	}
	defer pkgs[0].LPkg.Prog.Dispose()
	mod := pkgs[0].LPkg.Module()

	for _, name := range []string{"fatal", "throw"} {
		fn := mod.NamedFunction(llvmgen.PkgRuntime + "." + name)
		if fn.IsNil() {
			t.Fatalf("missing runtime function %s", name)
		}
		ir := fn.String()
		if !strings.Contains(ir, "noinline") {
			t.Fatalf("runtime.%s missing noinline attribute:\n%s", name, ir)
		}
	}
	exitFn := mod.NamedFunction(llvmgen.PkgRuntime + ".exitPanic")
	if exitFn.IsNil() {
		t.Fatal("missing runtime.exitPanic")
	}
	if ir := exitFn.String(); !strings.Contains(ir, "unreachable") {
		t.Fatalf("runtime.exitPanic missing unreachable terminator:\n%s", ir)
	}
	evacFn := mod.NamedFunction(llvmgen.PkgRuntime + ".evacuated")
	if evacFn.IsNil() {
		t.Fatal("missing runtime.evacuated")
	}
	if ir := evacFn.String(); strings.Contains(ir, "AssertNilDeref") {
		t.Fatalf("runtime.evacuated emitted unexpected AssertNilDeref:\n%s", ir)
	}
	recFn := mod.NamedFunction(llvmgen.PkgRuntime + ".Recover")
	if recFn.IsNil() {
		t.Fatal("missing runtime.Recover")
	}
	if ir := recFn.String(); !strings.Contains(ir, "noinline") {
		t.Fatalf("runtime.Recover missing noinline attribute:\n%s", ir)
	}
}
