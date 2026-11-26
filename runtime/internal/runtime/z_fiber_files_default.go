//go:build !llgo || !baremetal || !tinygo.riscv32 || nogc

package runtime

// Hosted and other baremetal targets compile no fiber helper objects.
const fiberLLGoFiles = ""
