//go:build !llgo
// +build !llgo

package ssa

import (
	"testing"

	"github.com/goplus/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

func TestTargetCodeGenOptLevel(t *testing.T) {
	tests := []struct {
		level optlevel.Level
		want  llvm.CodeGenOptLevel
	}{
		{level: optlevel.O0, want: llvm.CodeGenLevelNone},
		{level: optlevel.O1, want: llvm.CodeGenLevelLess},
		{level: optlevel.O2, want: llvm.CodeGenLevelDefault},
		{level: optlevel.O3, want: llvm.CodeGenLevelAggressive},
		{level: optlevel.Os, want: llvm.CodeGenLevelDefault},
		{level: optlevel.Oz, want: llvm.CodeGenLevelDefault},
	}
	for _, tt := range tests {
		target := &Target{OptLevel: tt.level}
		if got := target.codeGenOptLevel(); got != tt.want {
			t.Fatalf("codeGenOptLevel(%v) = %v, want %v", tt.level, got, tt.want)
		}
	}
}

func TestTargetDefaultOptLevel(t *testing.T) {
	if got := (&Target{}).effectiveOptLevel(); got != optlevel.O2 {
		t.Fatalf("host default opt level = %v, want %v", got, optlevel.O2)
	}
	if got := (&Target{Target: "rp2040"}).effectiveOptLevel(); got != optlevel.Oz {
		t.Fatalf("target default opt level = %v, want %v", got, optlevel.Oz)
	}
}
