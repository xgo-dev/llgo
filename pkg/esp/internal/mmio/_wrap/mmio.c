/* Volatile 32-bit register access for the pkg/esp baremetal backends.
 *
 * Go has no volatile: a load or store written in Go is fair game for LLVM to
 * merge (two reads of a status register become one) or to drop (a store made
 * dead by a later store to the same address). Every access here is spelled
 * `*(volatile uint32_t *)`, so each call is one bus access, in program order,
 * and nothing around it can be folded through it.
 *
 * llgoRiscv32Cycle is the tick source the drivers bound their waits with. It
 * duplicates the runtime's own csrr wrapper on purpose: the drivers must not
 * depend on a runtime-internal symbol, and the CSR read is three instructions.
 * A chip whose cycle counter stays frozen reports a constant value, so callers
 * pair this deadline with an iteration cap.
 */

#include <stdint.h>

uint32_t llgo_mmio_read32(uintptr_t addr)
{
    return *(volatile uint32_t *)addr;
}

void llgo_mmio_write32(uintptr_t addr, uint32_t value)
{
    *(volatile uint32_t *)addr = value;
}

void llgo_mmio_set32(uintptr_t addr, uint32_t mask)
{
    *(volatile uint32_t *)addr |= mask;
}

void llgo_mmio_clear32(uintptr_t addr, uint32_t mask)
{
    *(volatile uint32_t *)addr &= ~mask;
}

unsigned int llgo_mmio_cycles(void)
{
    unsigned int cycles;
    __asm__ volatile ("csrr %0, cycle" : "=r"(cycles));
    return cycles;
}
