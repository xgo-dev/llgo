package build

import (
	"slices"
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/crosscompile"
	llssa "github.com/goplus/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestConfigureWasmResume(t *testing.T) {
	t.Setenv(llgoWasmResume, "1")
	t.Setenv(llgoWasiThreads, "")
	export := crosscompile.Export{
		BuildTags: []string{"existing"},
		LDFLAGS:   []string{"before", "-sASYNCIFY=1", "after"},
		WasmPostLink: crosscompile.WasmPostLink{
			Asyncify:          true,
			TranslateToExnref: true,
		},
	}
	conf := &Config{Goos: "wasip1", Goarch: "wasm"}
	if err := configureWasmResume(conf, &export); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(export.BuildTags, wasmResumeBuildTag) {
		t.Fatalf("build tags = %v", export.BuildTags)
	}
	if export.WasmPostLink.Asyncify || slices.Contains(export.LDFLAGS, "-sASYNCIFY=1") {
		t.Fatalf("Asyncify remains enabled: %+v", export)
	}
	if !export.WasmPostLink.TranslateToExnref {
		t.Fatal("resumable WASI build disabled SjLj exception translation")
	}
	if err := configureWasmResume(conf, &export); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, tag := range export.BuildTags {
		if tag == wasmResumeBuildTag {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("resumable build tag count = %d, tags = %v", count, export.BuildTags)
	}
}

func TestConfigureWasmResumeRejectsUnsupportedModes(t *testing.T) {
	t.Setenv(llgoWasmResume, "1")
	for _, test := range []struct {
		name    string
		conf    Config
		threads bool
		want    string
	}{
		{name: "native", conf: Config{Goos: "linux", Goarch: "amd64"}, want: "requires GOARCH=wasm"},
		{name: "host", conf: Config{Goos: "linux", Goarch: "wasm"}, want: "does not support GOOS=linux"},
		{name: "threads", conf: Config{Goos: "wasip1", Goarch: "wasm"}, threads: true, want: llgoWasiThreads},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.threads {
				t.Setenv(llgoWasiThreads, "1")
			} else {
				t.Setenv(llgoWasiThreads, "")
			}
			err := configureWasmResume(&test.conf, &crosscompile.Export{})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("configureWasmResume error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestLowerWasmResumeModule(t *testing.T) {
	llvm.InitializeAllTargets()
	prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
	defer prog.Dispose()
	prog.EnableWasmResumeABI(true)
	pkg := prog.NewPackage("p", "example.com/p")
	callee := pkg.NewFunc("callee", llssa.NoArgsNoRet, llssa.InGo)
	callee.MakeBody(1).Return()
	caller := pkg.NewFunc("caller", llssa.NoArgsNoRet, llssa.InGo)
	b := caller.MakeBody(1)
	b.Call(callee.Expr)
	b.Return()

	ctx := &context{prog: prog}
	if err := lowerWasmResumeModule(ctx, pkg.Module()); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("verify lowered module: %v\n%s", err, pkg.String())
	}
	for _, want := range []string{
		"define internal i8 @__llgo_wasm_resume.caller",
		"define void @caller()",
		"define ptr @__llgo_wasm_start.caller",
	} {
		if !strings.Contains(pkg.String(), want) {
			t.Fatalf("lowered module is missing %q:\n%s", want, pkg.String())
		}
	}
}

func TestLowerWasmResumeModuleDisabled(t *testing.T) {
	if err := lowerWasmResumeModule(nil, llvm.Module{}); err != nil {
		t.Fatal(err)
	}
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	ctx := &context{prog: prog}
	if err := lowerWasmResumeModule(ctx, prog.NewPackage("p", "example.com/p").Module()); err != nil {
		t.Fatal(err)
	}
}

func TestLowerWasmResumeModuleReportsLoweringError(t *testing.T) {
	prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
	defer prog.Dispose()
	prog.EnableWasmResumeABI(true)
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("native")
	defer mod.Dispose()
	mod.SetTarget("aarch64-apple-darwin")

	err := lowerWasmResumeModule(&context{prog: prog}, mod)
	if err == nil || !strings.Contains(err.Error(), "target") {
		t.Fatalf("lowerWasmResumeModule error = %v", err)
	}
}
