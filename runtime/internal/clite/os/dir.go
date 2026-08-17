package os

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

type DIR struct {
	Unused [0]byte
}

//go:linkname Opendir C.opendir
func Opendir(name *c.Char) *DIR

//go:linkname Closedir C.closedir
func Closedir(dir *DIR) c.Int
