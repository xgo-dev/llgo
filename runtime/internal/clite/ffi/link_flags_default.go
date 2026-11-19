//go:build !wasm && !(baremetal && llgo.baremetal.stdio && go1.27)

package ffi

const (
	LLGoPackage = "link: $(pkg-config --libs libffi); -lffi"
	LLGoFiles   = "$(pkg-config --cflags libffi): _wrap/libffi.c"
)
