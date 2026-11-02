package main

import (
	_ "unsafe"

	"github.com/goplus/lib/c"
)

//go:linkname WriteByte C.board_uart_write_char
func WriteByte(b byte)

//go:linkname sleep C.sleep
func sleep(c c.Int)
