package build

import (
	"testing"

	"github.com/goplus/llgo/internal/optlevel"
)

func TestEffectiveOptLevelDefaults(t *testing.T) {
	t.Setenv(llgoOptimize, "")
	if got := effectiveOptLevel(&Config{}); got != optlevel.O2 {
		t.Fatalf("host default opt level = %v, want %v", got, optlevel.O2)
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
	if got := effectiveOptLevel(&Config{}); got != optlevel.O2 {
		t.Fatalf("LLGO_OPTIMIZE=off opt level = %v, want %v", got, optlevel.O2)
	}

	t.Setenv(llgoOptimize, "on")
	if got := effectiveOptLevel(&Config{}); got != optlevel.O2 {
		t.Fatalf("LLGO_OPTIMIZE=on opt level = %v, want %v", got, optlevel.O2)
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
		level optlevel.Level
		want  string
	}{
		{level: optlevel.O0, want: "default<O0>"},
		{level: optlevel.O1, want: "default<O1>"},
		{level: optlevel.O2, want: "default<O2>"},
		{level: optlevel.O3, want: "default<O3>"},
		{level: optlevel.Os, want: "default<Os>"},
		{level: optlevel.Oz, want: "default<Oz>"},
	}
	for _, tt := range tests {
		if got := llvmPassPipeline(tt.level); got != tt.want {
			t.Fatalf("llvmPassPipeline(%v) = %q, want %q", tt.level, got, tt.want)
		}
	}
}
