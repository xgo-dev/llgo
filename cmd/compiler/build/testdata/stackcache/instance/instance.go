package instance

import "github.com/xgo-dev/llgo/cmd/compiler/build/testdata/stackcache/generic"

func Value() int { return generic.Launch(42) }
