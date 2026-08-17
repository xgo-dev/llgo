//go:build !darwin
// +build !darwin

package os

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

//go:linkname Stat C.stat
func Stat(path *c.Char, buf *StatT) c.Int

//go:linkname Lstat C.lstat
func Lstat(path *c.Char, buf *StatT) c.Int

//go:linkname Fstat C.fstat
func Fstat(fd c.Int, buf *StatT) c.Int
