//go:build !llgo
// +build !llgo

package build

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"

	"github.com/goplus/llgo/internal/packages"
	llssa "github.com/goplus/llgo/ssa"
)

func init() {
	llssa.Initialize(llssa.InitAll)
}

func TestGenMainModuleExecutable(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	ctx := &context{
		prog: llssa.NewProgram(nil),
		buildConf: &Config{
			BuildMode: BuildModeExe,
			Goos:      "linux",
			Goarch:    "amd64",
		},
	}
	pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
	mod := genMainModule(ctx, llssa.PkgRuntime, pkg,
		&genConfig{rtInit: true, pyInit: true})
	if mod.ExportFile != "foo.a-main" {
		t.Fatalf("unexpected export file: %s", mod.ExportFile)
	}
	ir := mod.LPkg.String()
	checks := []string{
		"define i32 @main(",
		"call void @Py_Initialize()",
		"call void @\"example.com/foo.init\"()",
		"define weak void @_start()",
	}
	for _, want := range checks {
		if !strings.Contains(ir, want) {
			t.Fatalf("main module IR missing %q:\n%s", want, ir)
		}
	}
}

func TestGenMainModuleLibrary(t *testing.T) {
	llvm.InitializeAllTargets()
	t.Setenv(llgoStdioNobuf, "")
	ctx := &context{
		prog: llssa.NewProgram(nil),
		buildConf: &Config{
			BuildMode: BuildModeCArchive,
			Goos:      "linux",
			Goarch:    "amd64",
		},
	}
	pkg := &packages.Package{PkgPath: "example.com/foo", ExportFile: "foo.a"}
	mod := genMainModule(ctx, llssa.PkgRuntime, pkg, &genConfig{})
	ir := mod.LPkg.String()
	if strings.Contains(ir, "define i32 @main") {
		t.Fatalf("library mode should not emit main function:\n%s", ir)
	}
	if !strings.Contains(ir, "@__llgo_argc = global i32 0") {
		t.Fatalf("library mode missing argc global:\n%s", ir)
	}
}
