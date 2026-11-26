/* RV32 M-mode interrupt masking for the __atomic_* libcall shim in
 * atomic_riscv32.go. Go cannot express CSR instructions, so the shim's RMW
 * paths call these helpers.
 *
 * csrrc clears mstatus.MIE and returns the previous mstatus; the restore
 * writes only the MIE bit back with csrrs, so a caller that was already
 * running with interrupts masked stays masked and every other caller resumes
 * exactly as it left.
 *
 * Single-hart semantics: with interrupts masked, each guarded RMW sequence
 * is atomic against interrupt handlers on this hart. Nothing here protects
 * against other harts (the ESP32-C3/S31 baremetal runtime is single-hart) or
 * against non-maskable interrupts.
 */
unsigned int llgoAtomicIrqSave(void)
{
    unsigned int old;
    __asm__ volatile ("csrrc %0, mstatus, %1" : "=r"(old) : "r"(0x8u) : "memory");
    return old;
}

void llgoAtomicIrqRestore(unsigned int saved)
{
    __asm__ volatile ("csrrs zero, mstatus, %0" :: "r"(saved & 0x8u) : "memory");
}
