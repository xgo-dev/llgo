package os

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

//go:linkname Stat C.stat64
func Stat(path *c.Char, buf *StatT) c.Int

//go:linkname Lstat C.lstat64
func Lstat(path *c.Char, buf *StatT) c.Int

//go:linkname Fstat C.fstat64
func Fstat(fd c.Int, buf *StatT) c.Int
