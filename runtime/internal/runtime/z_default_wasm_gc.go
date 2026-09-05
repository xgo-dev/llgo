//go:build wasm && llgo.wasm.gc.linear

package runtime

import c "github.com/xgo-dev/llgo/runtime/internal/clite"

var (
	printFormatPrefixInt  = c.Str("%lld")
	printFormatPrefixUInt = c.Str("%llu")
	printFormatPrefixHex  = c.Str("%llx")
)
