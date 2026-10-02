package build

import (
	"testing"

	llruntime "github.com/xgo-dev/llgo/runtime"
	extplan9asm "github.com/xgo-dev/plan9asm"
)

func TestPlan9AsmTranslateOptions(t *testing.T) {
	tests := []struct {
		name string
		conf Config
		want extplan9asm.X87Mode
	}{
		{name: "386 default", conf: Config{Goarch: "386"}, want: extplan9asm.X87Auto},
		{name: "386 sse2", conf: Config{Goarch: "386", GO386: "sse2"}, want: extplan9asm.X87Auto},
		{name: "386 softfloat", conf: Config{Goarch: "386", GO386: "softfloat"}, want: extplan9asm.X87Software},
		{name: "other architecture", conf: Config{Goarch: "amd64", GO386: "softfloat"}, want: extplan9asm.X87Auto},
		{name: "amd64 v3", conf: Config{Goarch: "amd64", GOAMD64: "v3"}, want: extplan9asm.X87Auto},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := plan9asmTranslateOptions(&test.conf)
			if got.GOARM != test.conf.GOARM {
				t.Fatalf("GOARM = %q, want %q", got.GOARM, test.conf.GOARM)
			}
			if got.GOAMD64 != test.conf.GOAMD64 {
				t.Fatalf("GOAMD64 = %q, want %q", got.GOAMD64, test.conf.GOAMD64)
			}
			if got.X87Mode != test.want {
				t.Fatalf("X87Mode = %v, want %v", got.X87Mode, test.want)
			}
		})
	}
}

func TestInternalRuntimeSysUsesPlan9AsmWithoutAltPkg(t *testing.T) {
	conf := &Config{Goarch: "arm64"}
	if !plan9asmEnabledByDefault(conf, "internal/runtime/sys") {
		t.Fatal("plan9asm should be enabled by default for internal/runtime/sys on arm64")
	}
	if hasAltPkgForTarget(conf, "internal/runtime/sys") {
		t.Fatal("internal/runtime/sys should use its source patch instead of an alt package")
	}
}

func TestPlan9AsmDefaultsSupport386(t *testing.T) {
	conf := &Config{Goarch: "386"}
	for _, pkgPath := range []string{
		"internal/bytealg",
		"internal/chacha8rand",
		"internal/cpu",
		"internal/runtime/atomic",
		"internal/runtime/syscall/windows",
		"math",
	} {
		if !plan9asmEnabledByDefault(conf, pkgPath) {
			t.Errorf("plan9asm should be enabled by default for %s on 386", pkgPath)
		}
		if llruntime.SourcePatchReplacesAsmForGOARCH(pkgPath, "386") {
			t.Errorf("%s should retain the Go 386 assembly implementation", pkgPath)
		}
	}
	if !hasAltPkgForTarget(conf, "runtime") {
		t.Fatal("runtime should keep using the LLGo alternate package")
	}
}

func TestPlan9AsmDefaultsSupportWasm(t *testing.T) {
	conf := &Config{Goarch: "wasm"}
	for _, pkgPath := range []string{
		"crypto/internal/boring/sig",
		"internal/bytealg",
		"internal/chacha8rand",
		"internal/runtime/atomic",
		"math",
	} {
		if !plan9asmEnabledByDefault(conf, pkgPath) {
			t.Errorf("plan9asm should be enabled by default for %s on wasm", pkgPath)
		}
		if llruntime.SourcePatchReplacesAsmForGOARCH(pkgPath, "wasm") {
			t.Errorf("%s should retain the Go wasm assembly implementation", pkgPath)
		}
	}
}

func TestInternalRuntimeAtomicUsesSourcePatchOnArm(t *testing.T) {
	conf := &Config{Goarch: "arm"}
	if hasAltPkgForTarget(conf, "internal/runtime/atomic") {
		t.Fatal("internal/runtime/atomic should use its source patch on arm")
	}

	conf = &Config{Goarch: "arm64"}
	if hasAltPkgForTarget(conf, "internal/runtime/atomic") {
		t.Fatal("internal/runtime/atomic should keep plan9asm/std paths on arm64")
	}
}
