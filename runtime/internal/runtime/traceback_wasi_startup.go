//go:build llgo && wasip1 && llgo.wasi.mutatorselect

package runtime

import (
	"github.com/xgo-dev/llgo/runtime/internal/traceback"
)

func WASIStartupTraceback(data *byte, size uint32) uint32 {
	setting := traceback.Parse(wasmLiteralLocationString(data, int(size)), false)
	if setting&traceback.Crash != 0 {
		return 2
	}
	if traceback.Level(setting) != 0 {
		return 1
	}
	return 0
}
