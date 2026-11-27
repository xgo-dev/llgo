//go:build !(llgo && baremetal && !llgo.idf && (esp32c3 || esp32s31))

// Every build without a register window behind it: a hosted go build, an
// ESP-IDF build, and a baremetal target whose chip has no table in
// _wrap/mmio.c (the C accessors end in a RISC-V `csrr cycle`, so only the two
// supported SoCs can use this package at all). The one caller - the pkg/esp
// baremetal backends - is excluded by the same constraint, so every function
// here panics instead of inventing a register value: reaching one means the
// build tags of a caller are wrong.
package mmio

const noRegisterLayer = "esp/internal/mmio: no register layer in a hosted build"

// Read32 reports that register access does not exist in a hosted build.
func Read32(addr uintptr) uint32 {
	_ = addr
	panic(noRegisterLayer)
}

// Write32 reports that register access does not exist in a hosted build.
func Write32(addr uintptr, v uint32) {
	_, _ = addr, v
	panic(noRegisterLayer)
}

// Set32 reports that register access does not exist in a hosted build.
func Set32(addr uintptr, mask uint32) {
	_, _ = addr, mask
	panic(noRegisterLayer)
}

// Clear32 reports that register access does not exist in a hosted build.
func Clear32(addr uintptr, mask uint32) {
	_, _ = addr, mask
	panic(noRegisterLayer)
}

// Cycles reports that no cycle counter exists in a hosted build.
func Cycles() uint32 {
	panic(noRegisterLayer)
}
