package llvmgen_test

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/llvmgen"
	"github.com/xgo-dev/llgo/internal/llvmgen/ssatest"
)

func TestWasmImportAttributes(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	t.Cleanup(prog.Dispose)
	pkg := prog.NewPackage("foo", "foo")
	fn := pkg.NewFunc("fdRead", llvmgen.NoArgsNoRet, llvmgen.GoDecl)
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
