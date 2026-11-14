package build

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

const dispatchInlineIR = `
declare i32 @helper(ptr)
declare i32 @blocked(ptr) noinline
declare i32 @panic(ptr) cold noreturn
define void @dispatch(i32 %n, ptr %p, i1 %choose) {
entry:
  %outside = call i32 @helper(ptr %p)
  br i1 %choose, label %switch, label %other
switch:
  %header = call i32 @helper(ptr %p)
  switch i32 %n, label %exit [i32 0, label %work]
work:
  %body = call i32 @helper(ptr %p)
  %barrier = call i32 @blocked(ptr %p)
  %cold = call i32 @panic(ptr %p)
  %custom = call i32 @helper(ptr %p) "function-inline-threshold"="12"
  %explicit = call i32 @helper(ptr %p) noinline
  br label %switch
other:
  %unrelated = call i32 @helper(ptr %p)
  br i1 %choose, label %other, label %tail
tail:
  switch i32 %n, label %exit [i32 0, label %once]
once:
  %acyclic = call i32 @helper(ptr %p)
  ret void
exit:
  ret void
unreachable:
  %dead = call i32 @helper(ptr %p)
  switch i32 %n, label %unreachable [i32 0, label %exit]
}
`

func TestDispatchInlineBackendOverrides(t *testing.T) {
	for _, name := range []string{"CCFLAGS", "CFLAGS", "LDFLAGS"} {
		t.Setenv(name, "")
	}
	for _, test := range []struct {
		name   string
		change func(*context)
	}{
		{"default", nil},
		{"external linker", func(c *context) { c.buildConf.LinkOptions.ExternalLinker = "clang" }},
		{"direct linker", func(c *context) { c.crossCompile.Linker = "ld.lld" }},
		{"other backend", func(c *context) { c.crossCompile.Toolchain.Linker = "" }},
		{"extldflags", func(c *context) { c.buildConf.LinkOptions.ExternalLinkerFlags = "-Wl,-mllvm,-inline-threshold=20" }},
		{"driver args", func(c *context) { c.crossCompile.CXXArgs = []string{"-fno-inline"} }},
		{"target flags", func(c *context) { c.crossCompile.LDFLAGS = []string{"-Wl,-mllvm,-cold-callsite-rel-freq=0"} }},
		{"environment", func(c *context) { t.Setenv("LDFLAGS", "-Wl,-mllvm,-inline-cost-full=false") }},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := &context{buildConf: dispatchInlineConfig(), crossCompile: crosscompile.Export{
				Toolchain: crosscompile.NativeToolchain{ObjectFormat: crosscompile.ObjectFormatELF, Linker: crosscompile.LinkerFlavorELFLLD},
			}}
			if test.change != nil {
				test.change(c)
			}
			want := test.change == nil
			if got := c.dispatchInlineBudgetEnabled(); got != want {
				t.Fatalf("policy enabled = %v, want %v", got, want)
			}
			args := c.clangConfig().LDFLAGS
			found := false
			for _, arg := range args {
				found = found || arg == "-Wl,-mllvm,-inline-cost-full"
			}
			if found != want {
				t.Fatalf("complete-cost default = %v, want %v: %v", found, want, args)
			}
		})
	}
}

func dispatchInlineConfig() *Config {
	return &Config{Goos: "linux", Goarch: "amd64", BuildMode: BuildModeExe, OptLevel: optlevel.O3, LTO: lto.Full}
}

func TestDispatchInlineBudget(t *testing.T) {
	mod := parseWasmAggregateIR(t, dispatchInlineIR)
	if got := applyDispatchInlineBudget(dispatchInlineConfig(), mod); got != 2 {
		t.Fatalf("hinted %d calls, want 2", got)
	}
	for _, bb := range mod.NamedFunction("dispatch").BasicBlocks() {
		for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if inst.InstructionOpcode() != llvm.Call {
				continue
			}
			got := inst.GetCallSiteStringAttribute(-1, "function-inline-threshold")
			want := ""
			switch inst.Name() {
			case "header", "body":
				want = "250"
			case "custom":
				want = "12"
			}
			if want == "" {
				if !got.IsNil() {
					t.Errorf("%s unexpectedly has a budget", inst.Name())
				}
			} else if got.IsNil() || got.GetStringValue() != want {
				t.Errorf("%s budget = %v, want %s", inst.Name(), got, want)
			}
		}
	}
	if got := applyDispatchInlineBudget(dispatchInlineConfig(), mod); got != 0 {
		t.Fatalf("second application hinted %d calls", got)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestDispatchInlineBudgetScope(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*Config)
	}{
		{"O2", func(c *Config) { c.OptLevel = optlevel.O2 }},
		{"Os", func(c *Config) { c.OptLevel = optlevel.Os }},
		{"ThinLTO", func(c *Config) { c.LTO = lto.Thin }},
		{"off", func(c *Config) { c.LTO = lto.Off }},
		{"arm64", func(c *Config) { c.Goarch = "arm64" }},
		{"darwin", func(c *Config) { c.Goos = "darwin" }},
		{"library", func(c *Config) { c.BuildMode = BuildModeCArchive }},
	} {
		t.Run(test.name, func(t *testing.T) {
			mod := parseWasmAggregateIR(t, dispatchInlineIR)
			conf := dispatchInlineConfig()
			test.change(conf)
			if got := applyDispatchInlineBudget(conf, mod); got != 0 {
				t.Fatalf("hinted %d calls outside the scope", got)
			}
		})
	}
}

func TestDispatchInlineBudgetPreservesProfiles(t *testing.T) {
	source := strings.Replace(dispatchInlineIR, "br label %switch", "br label %switch, !prof !0", 1) + "\n!0 = !{!\"branch_weights\", i32 10}\n"
	mod := parseWasmAggregateIR(t, source)
	if got := applyDispatchInlineBudget(dispatchInlineConfig(), mod); got != 0 {
		t.Fatalf("hinted %d calls in a profiled function", got)
	}
}
