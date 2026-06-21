//go:build !llgo
// +build !llgo

package build

import (
	"testing"

	"github.com/goplus/llgo/internal/cabi"
)

func TestHasAltPkgForTarget_AllowsAdditivePatchWithPlan9Asm(t *testing.T) {
	conf := &Config{Goarch: "arm64", AbiMode: cabi.ModeAllFunc}
	if !plan9asmEnabledByDefault(conf, "internal/runtime/sys") {
		t.Fatal("plan9asm should be enabled by default for internal/runtime/sys on arm64")
	}
	if !hasAltPkgForTarget(conf, "internal/runtime/sys") {
		t.Fatal("internal/runtime/sys should keep its additive alt package even when plan9asm is enabled")
	}
}

func TestHasAltPkgForTarget_UsesAtomicFallbackOnArm(t *testing.T) {
	conf := &Config{Goarch: "arm", AbiMode: cabi.ModeAllFunc}
	if !hasAltPkgForTarget(conf, "internal/runtime/atomic") {
		t.Fatal("internal/runtime/atomic should use alt package on arm")
	}

	conf = &Config{Goarch: "arm64", AbiMode: cabi.ModeAllFunc}
	if hasAltPkgForTarget(conf, "internal/runtime/atomic") {
		t.Fatal("internal/runtime/atomic should keep plan9asm/std paths on arm64")
	}
}
