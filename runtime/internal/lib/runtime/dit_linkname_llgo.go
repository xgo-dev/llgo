//go:build go1.26

package runtime

import _ "unsafe"

// llgo does not currently model per-goroutine DIT state.
// Provide minimal linkname shims so crypto/subtle can build and
// fall back to its existing non-DIT behavior when unsupported.

//go:linkname dit_setEnabled crypto/subtle.setDITEnabled
func dit_setEnabled() bool {
	return false
}

//go:linkname dit_setDisabled crypto/subtle.setDITDisabled
func dit_setDisabled() {}
