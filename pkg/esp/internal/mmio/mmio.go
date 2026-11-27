//go:build llgo && baremetal && !llgo.idf && (esp32c3 || esp32s31)

// Package mmio is the volatile register-access layer the pkg/esp baremetal
// backends are built on.
//
// Go cannot express a volatile access: llgo has no //go:volatile marker, so a
// plain `*(*uint32)(unsafe.Pointer(addr))` reaches the backend as an ordinary
// non-volatile load or store. LLVM is then free to merge two reads of the same
// address into one (the accessor is inferred `readonly`, `nounwind` and
// `willreturn`, which makes it CSE-able) and to drop a store that a later
// store to the same address makes dead. Both transformations silently break
// hardware register access - a status-poll loop that reads the register once
// never terminates, a write-1-to-clear sequence loses a pulse.
//
// The accessors therefore live in _wrap/mmio.c, where `volatile` is
// expressible, and `LLGoFiles` pulls that file into any link that imports this
// package. The mechanism is not runtime-only: an ordinary peripheral package
// contributes its C files to the link just the same (see pkg/esp/README.md).
//
// Every function here performs exactly one bus access per call, in program
// order, and is safe to call from a spinning driver loop.
//
// # Choosing an accessor
//
// Write32 and Read32 are unconditional and are the only correct choice for a
// register the CPU cannot read meaningfully:
//
//   - write-1-to-set / write-1-to-clear registers (GPIO_OUT_W1TS/W1TC,
//     GPIO_ENABLE_W1TS/W1TC) and other write-only registers;
//   - self-clearing commands (I2C_SCL_SP_CONF's SCL_RST_SLV_EN, the TRM's
//     "write then wait for it to read back 0" pattern);
//   - interrupt clear registers (I2C_INT_CLR).
//
// Set32 and Clear32 are read-modify-write pairs, so they belong only on a
// readable read/write register: IO_MUX pad config, I2C CTR/FIFO_CONF, the
// clock-tree blocks. Using them on a write-only register reads back a value
// with no defined meaning, and what gets written is then either a no-op (the
// read returns 0) or damage to other bits (the read returns something else) -
// a write-1-to-clear register turns "clear one bit" into "clear every bit the
// read happened to set".
package mmio

import _ "unsafe" // for go:linkname

// LLGoFiles wires the volatile C accessors into the link.
const LLGoFiles = "_wrap/mmio.c"

// Read32 returns the 32-bit word at addr.
//
//go:linkname Read32 C.llgo_mmio_read32
func Read32(addr uintptr) uint32

// Write32 stores v at addr.
//
//go:linkname Write32 C.llgo_mmio_write32
func Write32(addr uintptr, v uint32)

// Set32 sets the bits in mask at addr as one read-modify-write pair.
//
//go:linkname Set32 C.llgo_mmio_set32
func Set32(addr uintptr, mask uint32)

// Clear32 clears the bits in mask at addr as one read-modify-write pair.
//
//go:linkname Clear32 C.llgo_mmio_clear32
func Clear32(addr uintptr, mask uint32)

// Cycles returns the low 32 bits of the RISC-V cycle counter, the free-running
// tick source the driver deadlines are measured against (160 MHz on both
// supported SoCs, so one tick is 6.25 ns). Callers must treat it as a bound
// only: a chip whose cycle counter stays frozen reports a constant value, so
// every deadline built on top of it needs an iteration cap as well.
//
//go:linkname Cycles C.llgo_mmio_cycles
func Cycles() uint32
