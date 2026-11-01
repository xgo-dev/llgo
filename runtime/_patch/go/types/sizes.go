package types

import (
	"runtime"
	"unsafe"
)

//llgo:cdecl
type llgoNativePointerProbe struct{ P unsafe.Pointer }

//llgo:cdecl
type llgoNativeAlignProbe struct{ F float64 }

// LLGo intentionally accepts both compiler identities. Neither name asks for
// cmd/compile storage when this package is built by LLGo.
func SizesFor(compiler, arch string) Sizes {
	switch compiler {
	case "gc", "llgo":
		if s := llgoArchSizes[arch]; s != nil {
			return s
		}
	case "gccgo":
		if s, ok := gccgoArchSizes[arch]; ok {
			return s
		}
	}
	return nil
}

var llgoSourceLayout LLGoLayout

// Keep the stable factory identity used by go/packages metadata drivers.
// Checker.Files copies a Sizes value before attaching per-checker settings.
var llgoArchSizes = func() map[string]*LLGoSizes {
	out := make(map[string]*LLGoSizes)
	for _, arch := range []string{"386", "amd64", "amd64p32", "arm", "arm64", "loong64", "mips", "mipsle", "mips64", "mips64le", "ppc64", "ppc64le", "riscv", "riscv64", "s390x", "sparc64", "wasm", "wasm64"} {
		s := LLGoSizesFor(arch)
		if arch == runtime.GOARCH {
			// GOARCH alone cannot distinguish wasm Memory32/Memory64 or
			// the platform-dependent C double alignment on 386.
			s.WordSize = int64(unsafe.Sizeof(uintptr(0)))
			s.MaxAlign = int64(unsafe.Alignof(complex128(0)))
			s.PointerSize = int64(unsafe.Sizeof(llgoNativePointerProbe{}))
			s.NativeAlign64 = int64(unsafe.Alignof(llgoNativeAlignProbe{}))
		}
		s.Layout = &llgoSourceLayout
		out[arch] = s
	}
	return out
}()
