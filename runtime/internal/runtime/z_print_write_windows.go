//go:build windows

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
)

const platformLLGoFiles = "; _wrap/print_windows.c; _wrap/rand_windows.c"

//go:linkname c_printByte C.llgo_print_byte
func c_printByte(v byte)

//go:linkname c_printWrite C.llgo_print_write
func c_printWrite(data unsafe.Pointer, size uintptr)

//go:linkname c_printStderrIsConsole C.llgo_print_stderr_is_console
func c_printStderrIsConsole() c.Int

//go:linkname c_printWriteConsole C.llgo_print_write_console
func c_printWriteConsole(data *uint16, size uintptr)

var (
	windowsConsoleBuffer [1000]uint16
	windowsConsoleMu     psync.Mutex
)

func PrintByte(v byte) {
	c_printByte(v)
}

func PrintString(s String) {
	text := *(*string)(unsafe.Pointer(&s))
	for i := 0; i < len(text); i++ {
		if text[i] >= runeSelf {
			if c_printStderrIsConsole() != 0 {
				printWindowsConsole(text)
				return
			}
			break
		}
	}
	c_printWrite(s.data, uintptr(s.len))
}

// printWindowsConsole follows the Go runtime's Windows console path: use
// WriteConsoleW for non-ASCII output so the result does not depend on the
// active console code page. Keep the conversion buffer static and avoid defer;
// this path is also used while reporting panics.
func printWindowsConsole(text string) {
	const surrogateOffset = (surrogateMin + surrogateMax + 1) / 2

	windowsConsoleMu.Lock()
	buffer := windowsConsoleBuffer[:]
	written := 0
	for i := 0; i < len(text); {
		if written >= len(buffer)-2 {
			c_printWriteConsole(&buffer[0], uintptr(written))
			written = 0
		}
		r := rune(text[i])
		if r < runeSelf {
			i++
		} else {
			r, i = decodeNonASCII(text, i)
		}
		if r < 0x10000 {
			buffer[written] = uint16(r)
			written++
		} else {
			r -= 0x10000
			buffer[written] = surrogateMin + uint16(r>>10)&0x3ff
			buffer[written+1] = surrogateOffset + uint16(r)&0x3ff
			written += 2
		}
	}
	if written != 0 {
		c_printWriteConsole(&buffer[0], uintptr(written))
	}
	windowsConsoleMu.Unlock()
}
