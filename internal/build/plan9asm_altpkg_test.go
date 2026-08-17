//go:build !llgo
// +build !llgo

package build

import (
	"testing"

	"github.com/xgo-dev/llgo/internal/cabi"
)

func TestInternalRuntimeSysUsesPlan9AsmWithoutAltPkg(t *testing.T) {
	conf := &Config{Goarch: "arm64", AbiMode: cabi.ModeAllFunc}
	if !plan9asmEnabledByDefault(conf, "internal/runtime/sys") {
		t.Fatal("plan9asm should be enabled by default for internal/runtime/sys on arm64")
	}
	if hasAltPkgForTarget(conf, "internal/runtime/sys") {
		t.Fatal("internal/runtime/sys should use its source patch instead of an alt package")
	}
}

func TestInternalRuntimeAtomicUsesSourcePatchOnArm(t *testing.T) {
	conf := &Config{Goarch: "arm", AbiMode: cabi.ModeAllFunc}
	if hasAltPkgForTarget(conf, "internal/runtime/atomic") {
		t.Fatal("internal/runtime/atomic should use its source patch on arm")
	}

	conf = &Config{Goarch: "arm64", AbiMode: cabi.ModeAllFunc}
	if hasAltPkgForTarget(conf, "internal/runtime/atomic") {
		t.Fatal("internal/runtime/atomic should keep plan9asm/std paths on arm64")
	}
}
