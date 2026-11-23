//go:build llgo && baremetal && llgo.embedded.startup

package runtime

import _ "unsafe"

const EmbeddedStartupProof = true

// RequireEmbeddedStartup is resolved by the normal entry. If the compact
// link retains a runtime operation that depends on startup, retry that link
// with the ordinary entry and the same archives.
//
//go:linkname RequireEmbeddedStartup C.__llgo_embedded_requires_runtime_startup
func RequireEmbeddedStartup()
