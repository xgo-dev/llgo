package build

import (
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

func TestEffectiveOptLevelDefaults(t *testing.T) {
	t.Setenv(llgoOptimize, "")
	if got := effectiveOptLevel(&Config{}); got != optlevel.Oz {
		t.Fatalf("host default opt level = %v, want %v", got, optlevel.Oz)
	}
	if got := effectiveOptLevel(&Config{Target: "rp2040"}); got != optlevel.Oz {
		t.Fatalf("target default opt level = %v, want %v", got, optlevel.Oz)
	}
}

func TestEffectiveOptLevelOverride(t *testing.T) {
	if got := effectiveOptLevel(&Config{OptLevel: optlevel.O3}); got != optlevel.O3 {
		t.Fatalf("explicit opt level = %v, want %v", got, optlevel.O3)
	}
	if got := effectiveOptLevel(&Config{Target: "rp2040", OptLevel: optlevel.Os}); got != optlevel.Os {
		t.Fatalf("explicit target opt level = %v, want %v", got, optlevel.Os)
	}
}

func TestEffectiveOptLevelIgnoresLegacyEnv(t *testing.T) {
	t.Setenv(llgoOptimize, "off")
	if got := effectiveOptLevel(&Config{}); got != optlevel.Oz {
		t.Fatalf("LLGO_OPTIMIZE=off opt level = %v, want %v", got, optlevel.Oz)
	}

	t.Setenv(llgoOptimize, "on")
	if got := effectiveOptLevel(&Config{}); got != optlevel.Oz {
		t.Fatalf("LLGO_OPTIMIZE=on opt level = %v, want %v", got, optlevel.Oz)
	}
}

func TestApplySizeOptimizationAttributes(t *testing.T) {
	tests := []struct {
		level       optlevel.Level
		wantOptSize bool
		wantMinSize bool
	}{
		{level: optlevel.O2},
		{level: optlevel.Os, wantOptSize: true},
		{level: optlevel.Oz, wantOptSize: true, wantMinSize: true},
	}
	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			mod := ctx.NewModule("size-attributes")
			defer mod.Dispose()

			fnType := llvm.FunctionType(ctx.VoidType(), nil, false)
			defined := llvm.AddFunction(mod, "defined", fnType)
			block := ctx.AddBasicBlock(defined, "entry")
			builder := ctx.NewBuilder()
			builder.SetInsertPointAtEnd(block)
			builder.CreateRetVoid()
			builder.Dispose()
			declared := llvm.AddFunction(mod, "declared", fnType)

			applySizeOptimizationAttributes(mod, tt.level)
			optSizeKind := llvm.AttributeKindID("optsize")
			minSizeKind := llvm.AttributeKindID("minsize")
			if got := !defined.GetEnumFunctionAttribute(optSizeKind).IsNil(); got != tt.wantOptSize {
				t.Fatalf("defined optsize = %v, want %v", got, tt.wantOptSize)
			}
			if got := !defined.GetEnumFunctionAttribute(minSizeKind).IsNil(); got != tt.wantMinSize {
				t.Fatalf("defined minsize = %v, want %v", got, tt.wantMinSize)
			}
			if !declared.GetEnumFunctionAttribute(optSizeKind).IsNil() ||
				!declared.GetEnumFunctionAttribute(minSizeKind).IsNil() {
				t.Fatal("size attributes should not be added to declarations")
			}
		})
	}
}

func TestIsOptimizeEnabledLegacyEnv(t *testing.T) {
	t.Setenv(llgoOptimize, "off")
	if IsOptimizeEnabled() {
		t.Fatal("LLGO_OPTIMIZE=off should disable legacy optimize switch")
	}

	t.Setenv(llgoOptimize, "on")
	if !IsOptimizeEnabled() {
		t.Fatal("LLGO_OPTIMIZE=on should enable legacy optimize switch")
	}
}

func TestLLVMPassPipeline(t *testing.T) {
	tests := []struct {
		level   optlevel.Level
		ltoMode lto.Mode
		want    string
	}{
		{level: optlevel.O0, want: "default<O0>"},
		{level: optlevel.O1, want: "default<O1>"},
		{level: optlevel.O2, want: "default<O2>"},
		{level: optlevel.O3, want: "default<O3>"},
		{level: optlevel.Os, want: "default<Os>"},
		{level: optlevel.Oz, want: "default<Oz>"},
		{level: optlevel.O2, ltoMode: lto.Full, want: "lto-pre-link<O2>"},
		{level: optlevel.Oz, ltoMode: lto.Thin, want: "thinlto-pre-link<Oz>"},
	}
	for _, tt := range tests {
		if got := llvmPassPipeline(tt.level, tt.ltoMode); got != tt.want {
			t.Fatalf("llvmPassPipeline(%v, %v) = %q, want %q", tt.level, tt.ltoMode, got, tt.want)
		}
	}
}

func TestShouldRunLLVMPasses(t *testing.T) {
	for _, mode := range []Mode{ModeBuild, ModeInstall, ModeRun, ModeTest, ModeCmpTest} {
		if !shouldRunLLVMPasses(mode) {
			t.Errorf("shouldRunLLVMPasses(%v) = false, want true", mode)
		}
	}
	if shouldRunLLVMPasses(ModeGen) {
		t.Fatal("shouldRunLLVMPasses(ModeGen) = true, want false")
	}
}
