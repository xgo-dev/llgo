//go:build go1.27 && !windows

package runtime

import _ "unsafe"

// Go 1.27's crypto FIPS helpers link to this runtime monotonic-clock hook.
// LLGo already exposes the same clock through runtimeNano.
//
//go:linkname crypto_internal_fips140deps_time_monoTime crypto/internal/fips140deps/time.monoTime
func crypto_internal_fips140deps_time_monoTime() int64 {
	return runtimeNano()
}
