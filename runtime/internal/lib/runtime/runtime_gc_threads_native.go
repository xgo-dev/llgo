//go:build !nogc && !baremetal && !wasm && (windows || darwin || linux)

package runtime

import "github.com/xgo-dev/llgo/runtime/internal/clite/bdwgc"

func enableForeignThreadRegistration() {
	bdwgc.AllowRegisterThreads()
}
