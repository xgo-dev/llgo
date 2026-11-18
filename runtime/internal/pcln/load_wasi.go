//go:build llgo && llgo_pclntab_external && wasm && wasip1

package pcln

import "unsafe"

const externalPCLNGOOS = 3
const LLGoFiles = "_wrap/load_wasi.c"

//go:linkname load C.llgo_pcln_load
func load(out *unsafe.Pointer, size *uint32)

func readExternalPCLN(_ func(ready func() bool)) []byte {
	var ptr unsafe.Pointer
	var size uint32
	load(&ptr, &size)
	return externalBuffer(ptr, size)
}
