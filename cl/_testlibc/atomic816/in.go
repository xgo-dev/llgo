// 1/2-byte atomic coverage: the width-generic llgo.atomic* intrinsics take
// their operand width from the argument type, so linknaming them with
// uint8/uint16 operands emits atomicrmw i8/i16. On RV32IMC baremetal targets
// without the A extension, LLVM AtomicExpand lowers those to the
// __atomic_*_1/_2 libcalls served by runtime/internal/runtime/atomic_riscv32.go
// (with the vendored rtlib archive as the no-runtime fallback), so this
// fixture exercises that shim family end-to-end on every backend.
//
// Fetch ops return the previous value and store the combined result; a
// failing CAS must write the current value back into the expected slot (the
// AtomicExpand-served read-back contract). Output uses println, whose
// integer formatting is identical hosted and baremetal, so the golden is
// backend-independent.
package main

import _ "unsafe"

// llgo:link atomicLoad8 llgo.atomicLoad
func atomicLoad8(p *uint8) uint8 { return 0 }

// llgo:link atomicStore8 llgo.atomicStore
func atomicStore8(p *uint8, v uint8) {}

// llgo:link atomicExchange8 llgo.atomicXchg
func atomicExchange8(p *uint8, v uint8) uint8 { return 0 }

// llgo:link atomicCmpXchg8 llgo.atomicCmpXchg
func atomicCmpXchg8(p *uint8, old, new uint8) (uint8, bool) { return 0, false }

// llgo:link atomicAdd8 llgo.atomicAdd
func atomicAdd8(p *uint8, v uint8) uint8 { return 0 }

// llgo:link atomicSub8 llgo.atomicSub
func atomicSub8(p *uint8, v uint8) uint8 { return 0 }

// llgo:link atomicAnd8 llgo.atomicAnd
func atomicAnd8(p *uint8, v uint8) uint8 { return 0 }

// llgo:link atomicOr8 llgo.atomicOr
func atomicOr8(p *uint8, v uint8) uint8 { return 0 }

// llgo:link atomicXor8 llgo.atomicXor
func atomicXor8(p *uint8, v uint8) uint8 { return 0 }

// llgo:link atomicNand8 llgo.atomicNand
func atomicNand8(p *uint8, v uint8) uint8 { return 0 }

// llgo:link atomicLoad16 llgo.atomicLoad
func atomicLoad16(p *uint16) uint16 { return 0 }

// llgo:link atomicStore16 llgo.atomicStore
func atomicStore16(p *uint16, v uint16) {}

// llgo:link atomicExchange16 llgo.atomicXchg
func atomicExchange16(p *uint16, v uint16) uint16 { return 0 }

// llgo:link atomicCmpXchg16 llgo.atomicCmpXchg
func atomicCmpXchg16(p *uint16, old, new uint16) (uint16, bool) { return 0, false }

// llgo:link atomicAdd16 llgo.atomicAdd
func atomicAdd16(p *uint16, v uint16) uint16 { return 0 }

// llgo:link atomicSub16 llgo.atomicSub
func atomicSub16(p *uint16, v uint16) uint16 { return 0 }

// llgo:link atomicAnd16 llgo.atomicAnd
func atomicAnd16(p *uint16, v uint16) uint16 { return 0 }

// llgo:link atomicOr16 llgo.atomicOr
func atomicOr16(p *uint16, v uint16) uint16 { return 0 }

// llgo:link atomicXor16 llgo.atomicXor
func atomicXor16(p *uint16, v uint16) uint16 { return 0 }

// llgo:link atomicNand16 llgo.atomicNand
func atomicNand16(p *uint16, v uint16) uint16 { return 0 }

var (
	b8  uint8
	w16 uint16
)

func main() {
	// uint8 walk: 250 -> 255 -> 0 -> 255 -> 15 -> 14 -> 247 -> 254 -> 253
	atomicStore8(&b8, 250)
	println("load8", atomicLoad8(&b8) == 250)
	println("or8", atomicOr8(&b8, 5) == 250, b8 == 255)
	println("add8", atomicAdd8(&b8, 1) == 255, b8 == 0)
	println("sub8", atomicSub8(&b8, 1) == 0, b8 == 255)
	println("and8", atomicAnd8(&b8, 15) == 255, b8 == 15)
	println("xor8", atomicXor8(&b8, 1) == 15, b8 == 14)
	println("nand8", atomicNand8(&b8, 9) == 14, b8 == 247)
	println("exchange8", atomicExchange8(&b8, 254) == 247, b8 == 254)

	old, ok := atomicCmpXchg8(&b8, 254, 253)
	println("cas8-hit", ok, old == 254, b8 == 253)
	old, ok = atomicCmpXchg8(&b8, 1, 99)
	println("cas8-miss", !ok, old == 253, b8 == 253)

	// uint16 walk: 65530 -> 65535 -> 0 -> 65535 -> 255 -> 0 -> 65535 -> 65534 -> 65533
	atomicStore16(&w16, 65530)
	println("load16", atomicLoad16(&w16) == 65530)
	println("or16", atomicOr16(&w16, 5) == 65530, w16 == 65535)
	println("add16", atomicAdd16(&w16, 1) == 65535, w16 == 0)
	println("sub16", atomicSub16(&w16, 1) == 0, w16 == 65535)
	println("and16", atomicAnd16(&w16, 255) == 65535, w16 == 255)
	println("xor16", atomicXor16(&w16, 255) == 255, w16 == 0)
	println("nand16", atomicNand16(&w16, 15) == 0, w16 == 65535)
	println("exchange16", atomicExchange16(&w16, 65534) == 65535, w16 == 65534)

	old16, ok16 := atomicCmpXchg16(&w16, 65534, 65533)
	println("cas16-hit", ok16, old16 == 65534, w16 == 65533)
	old16, ok16 = atomicCmpXchg16(&w16, 1, 99)
	println("cas16-miss", !ok16, old16 == 65533, w16 == 65533)
}
