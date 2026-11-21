/* Fiber context switch for RV32IMC baremetal targets (ESP32-C3/S31).
 *
 * llgoFiberSwap saves the calling context's ra and all twelve RV32I
 * callee-saved integer registers (s0-s11 = x8,x9,x18-x27; the ilp32 soft-float
 * ABI affects only the f registers, so every s register is live here) in a
 * 64-byte frame below its entry stack pointer, publishes that frame address in
 * from->sp, then loads to->sp and its saved frame. Resuming a context is a
 * plain ret into the restored ra, so no atomic instructions are involved and
 * the switch runs on plain RV32IMC load/store code.
 *
 * The frame layout must stay in sync with the fiberFrame constants in
 * proc_fiber_baremetal.go, which writes a new fiber's initial frame. The
 * frame size preserves the psABI 16-byte stack alignment; a0 is saved as the
 * first-entry argument slot even though ordinary switches do not carry a0
 * across. The initial frame only fills ra and a0; the s0-s11 slots stay at
 * their allocator-zeroed contents. That is safe because the entry stub never
 * returns through this frame: the restored s0-s11 are meaningless scratch
 * values that the stub's own code overwrites, and the first save of the
 * fiber's registers happens on a later, fully-written swap frame.
 *
 * Two scheduler contracts this code relies on:
 *   - Interrupt handlers must not touch scheduler state (run queues, G
 *     statuses, fiber contexts). The runtime on these targets dispatches no
 *     Go code from ISRs, so llgoFiberSwap never runs concurrently with
 *     itself; a swap is only safe at cooperative yield points.
 *   - Between "sw sp, 0(a0)" and "lw sp, 0(a1)" the context is neither fully
 *     on the old nor the new stack (a non-atomic window). Nothing may read
 *     from->sp or to->sp during that window: the swap is single-hart code
 *     with interrupts masked by the surrounding scheduler logic's implicit
 *     assumption that no ISR re-enters it.
 *
 * llgoFiberGoStart and llgoFiberMainStart are //export-ed Go functions
 * (proc_fiber_baremetal.go). The address helpers hand their entry points to
 * the scheduler without exposing Go func-value representation, and keep the
 * exported symbols referenced so --gc-sections cannot drop them.
 */

extern void *llgoFiberGoStart(void *arg);
extern void *llgoFiberMainStart(void *arg);

void *llgoFiberGStartAddr(void)
{
    return (void *)llgoFiberGoStart;
}

void *llgoFiberMainStartAddr(void)
{
    return (void *)llgoFiberMainStart;
}

__attribute__((naked)) void llgoFiberSwap(unsigned int *fromSp, unsigned int *toSp)
{
    __asm__ volatile(
        "addi sp, sp, -64\n"
        "sw ra, 0(sp)\n"
        "sw s0, 4(sp)\n"
        "sw s1, 8(sp)\n"
        "sw s2, 12(sp)\n"
        "sw s3, 16(sp)\n"
        "sw s4, 20(sp)\n"
        "sw s5, 24(sp)\n"
        "sw s6, 28(sp)\n"
        "sw s7, 32(sp)\n"
        "sw s8, 36(sp)\n"
        "sw s9, 40(sp)\n"
        "sw s10, 44(sp)\n"
        "sw s11, 48(sp)\n"
        "sw a0, 52(sp)\n"
        "sw sp, 0(a0)\n"
        "lw sp, 0(a1)\n"
        "lw ra, 0(sp)\n"
        "lw s0, 4(sp)\n"
        "lw s1, 8(sp)\n"
        "lw s2, 12(sp)\n"
        "lw s3, 16(sp)\n"
        "lw s4, 20(sp)\n"
        "lw s5, 24(sp)\n"
        "lw s6, 28(sp)\n"
        "lw s7, 32(sp)\n"
        "lw s8, 36(sp)\n"
        "lw s9, 40(sp)\n"
        "lw s10, 44(sp)\n"
        "lw s11, 48(sp)\n"
        "lw a0, 52(sp)\n"
        "addi sp, sp, 64\n"
        "ret\n");
}
