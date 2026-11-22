//go:build !llgo || !baremetal || !llgo.embedded.startup

package c

const EmbeddedStartupProof = false

func RequireEmbeddedStartup() {}
