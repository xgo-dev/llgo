//go:build go1.26

package runtime

import (
	_ "unsafe"

	"github.com/goplus/llgo/runtime/internal/clite/tls"
)

// llgo does not currently expose runtime.getg in this compatibility layer.
// Its goroutines do not migrate between OS threads, so pthread TLS gives the
// bypass state the same non-process-wide isolation that crypto/fips140 needs.
var fipsBypassTLS = tls.Alloc[uint32](nil)

//go:linkname fips140_setBypass crypto/fips140.setBypass
func fips140_setBypass() {
	fipsBypassTLS.Set(fipsBypassTLS.Get() + 1)
}

//go:linkname fips140_unsetBypass crypto/fips140.unsetBypass
func fips140_unsetBypass() {
	if n := fipsBypassTLS.Get(); n != 0 {
		fipsBypassTLS.Set(n - 1)
	}
}

//go:linkname fips140_isBypassed crypto/fips140.isBypassed
func fips140_isBypassed() bool {
	return fipsBypassTLS.Get() != 0
}
