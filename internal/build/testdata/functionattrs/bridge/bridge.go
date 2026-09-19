package bridge

import "github.com/xgo-dev/llgo/internal/build/testdata/functionattrs/dep"

//go:noinline
func Stop() { dep.Stop() }
