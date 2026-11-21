/* Monotonic clock source for the RV32IMC fiber-scheduler baremetal targets
 * (ESP32-C3/S31). Go cannot express CSR instructions, so the cycle counter
 * read lives here and links into the runtime through the LLGoFiles chain in
 * z_rt.go (z_clock_files_fiber.go pairs the constant).
 *
 * csrr cycle returns the low 32 bits of the mcycle counter; RV32 can observe
 * only one half at a time, so all wrap handling lives on the Go side
 * (clock_fiber_baremetal.go). The cycle CSR is a base RISC-V register that
 * every real RV32 core implements, but esp-emu's esp32c3 model implements no
 * clock at all — that is why the C3 emulation suite runs no time fixtures
 * while the esp32s31 model (and real hardware) does.
 */
unsigned int llgoRiscv32Cycle(void)
{
    unsigned int cycles;
    __asm__ volatile ("csrr %0, cycle" : "=r"(cycles));
    return cycles;
}
