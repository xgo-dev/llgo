//go:build !llgo || !baremetal || !llgo.embedded.startup

package runtime

const EmbeddedStartupProof = false

func RequireEmbeddedStartup() {}
