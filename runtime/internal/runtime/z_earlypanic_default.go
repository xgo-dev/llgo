//go:build !llgo || !baremetal || !tinygo.riscv32 || !esp || llgo.idf || nogc

package runtime

// Hosted and the baremetal targets without the ESP USB-Serial/JTAG console
// compile no early-panic helpers: they keep the plain stdio-backed panic
// path. The complement of earlypanic_baremetal.go's build tag.
func earlyPanicReport(any) {}

const earlyPanicLLGoFiles = ""
