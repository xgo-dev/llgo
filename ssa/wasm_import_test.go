//go:build !llgo

package ssa_test

import (
	"strings"
	"testing"

	"github.com/goplus/llgo/ssa"
	"github.com/goplus/llgo/ssa/ssatest"
)

func TestWasmImportAttributes(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")
	fn := pkg.NewFunc("fdRead", ssa.NoArgsNoRet, ssa.InGo)
	fn.SetWasmImport("wasi_snapshot_preview1", "fd_read")

	ir := pkg.Module().String()
	for _, want := range []string{
		`"wasm-import-module"="wasi_snapshot_preview1"`,
		`"wasm-import-name"="fd_read"`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing %s in wasm import IR:\n%s", want, ir)
		}
	}
}
