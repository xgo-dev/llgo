package impl

import (
	"github.com/xgo-dev/llgo/internal/build/testdata/crossinlinedecl/decl"
	_ "unsafe"
)

//go:linkname value github.com/xgo-dev/llgo/internal/build/testdata/crossinlinedecl/decl.Number.Value
func value(n decl.Number) int64 { return int64(n) + 1 }
