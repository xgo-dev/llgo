//go:build llgo.baremetal.directconsole

package runtime

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const platformLLGoFiles = "; _wrap/print_baremetal.c"

//go:linkname directBuiltinConsole C.__llgo_baremetal_direct_console
var directBuiltinConsole bool

//go:linkname writeConsoleByte C.llgo_baremetal_console_byte
func writeConsoleByte(byte)

//go:linkname writeConsole C.llgo_baremetal_console_write
func writeConsole(c.Pointer, c.SizeT)

func PrintByte(v byte) {
	if directBuiltinConsole {
		writeConsoleByte(v)
		return
	}
	c.Fputc(c.Int(v), c.Stderr)
}

func PrintString(s String) {
	if directBuiltinConsole {
		writeConsole(s.data, c.SizeT(s.len))
		return
	}
	c.Fwrite(s.data, 1, uintptr(s.len), c.Stderr)
}
