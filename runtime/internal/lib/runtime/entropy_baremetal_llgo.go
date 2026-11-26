//go:build llgo && baremetal && linux && arm && !llgo.idf

package runtime

import _ "unsafe" // for go:linkname

// llgoEntropyFill is the sysrand boundary behind crypto/rand.Read on
// baremetal targets (see runtime/_patch/crypto/internal/sysrand). Without
// the ESP-IDF bridge there is no known entropy source on these targets, so
// the fill reports failure and crypto/rand.Read dies with the standard
// library's honest fatal message instead of returning bytes that only look
// random. Wiring a chip-specific TRNG (register-level or via a future
// platform bridge) means adding a sibling file constrained to that target.
//
//go:linkname llgoEntropyNoFill runtime.llgoEntropyFill
func llgoEntropyNoFill(b []byte) bool {
	return false
}
