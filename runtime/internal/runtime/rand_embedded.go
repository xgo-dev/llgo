//go:build llgo && baremetal && llgo.embedded.startup && !windows

package runtime

import c "github.com/xgo-dev/llgo/runtime/internal/clite"

func fastrand() uint32 {
	RequireEmbeddedStartup()
	return uint32(c.Rand())
}
