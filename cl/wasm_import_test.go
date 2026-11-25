package cl_test

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cl/cltest"
	"github.com/xgo-dev/llgo/internal/llvmgen"
)

func TestWasmImportDirective(t *testing.T) {
	const src = `package foo

//go:wasmimport wasi_snapshot_preview1 fd_read
func fdRead(fd int32, buf *byte, size uint32) uint32

func read(buf *byte) uint32 {
	return fdRead(0, buf, 1)
}
`
	ir := cltest.CompileIREx(t, src, "foo.go", false, func(prog llvmgen.Program) {
		prog.Target().GOOS = "wasip1"
		prog.Target().GOARCH = "wasm"
	})
	for _, want := range []string{
		`"wasm-import-module"="wasi_snapshot_preview1"`,
		`"wasm-import-name"="fd_read"`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing %s in wasm import IR:\n%s", want, ir)
		}
	}
}
