//go:build !baremetal

package runtime

import c "github.com/goplus/llgo/runtime/internal/clite"

var (
	printFormatPrefixInt  = c.Str("%lld")
	printFormatPrefixUInt = c.Str("%llu")
	printFormatPrefixHex  = c.Str("%llx")
)
