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
 *
 * Stack-overflow guard
 *
 * A fiber stack is one contiguous tinygogc allocation (Alloc returns a single
 * run of blocks inside one heap segment), bounded by the stackLo/stackHi
 * fields of its baremetalFiberContext. The Go side stores those bounds per
 * fiber (and the bootstrap scheduler bounds in the system context) and marks
 * the floor word of every stack with fiberStackCanary (proc_fiber_baremetal.go).
 * Before pushing its own 64-byte frame, llgoFiberSwap checks the outgoing
 * context: the canary word at from->stackLo (offset 0 of the stack) must be
 * intact and the live sp must leave room for the swap frame above stackLo.
 * This mirrors TinyGo, whose task.Pause validates a stack canary at exactly
 * the cooperative switch point and reports "goroutine stack overflow"; a
 * function-prologue SP check would need a compiler pass that llgo's cl/
 * pipeline does not have (see the TODO in proc_fiber_baremetal.go).
 *
 * Consequences of that checkpoint granularity, shared with TinyGo's Pause
 * semantics: writes below the floor are detected at the next yield, after
 * they happened - the guard bounds the damage to at most one call frame of
 * overshoot (the deepest Go frame below stackLo plus the swap frame), and
 * it stops the program deterministically instead of letting it keep running
 * on a smashed stack. A fiber whose sp recovered before the yield is caught
 * by the canary. A recursion that never yields is not caught.
 */

/* fiberStackCanary: floor-word magic shared with proc_fiber_baremetal.go.
 * The allocator zeroes new stacks, so any other value at the floor word
 * proves a write crossed the stack floor. The stringize indirection keeps
 * the literal in one place; the assembler takes the bare hex digits. */
#define llgoFiberStr2(x) #x
#define llgoFiberStr(x) llgoFiberStr2(x)
#define llgoFiberStackCanary 0x5afec0de

/* Fiber swap-frame geometry, mirrored by the Go constants in
 * proc_fiber_baremetal.go (fiberFrameSize 64, fiberFrameRAOff 0,
 * fiberFrameArgOff 52) and pinned by the compile-time assertions in
 * fiber_guard_layout.go. The raw offsets in the asm below must stay equal to
 * these macros; the static asserts fail the C build on any drift. */
#define llgoFiberFrameSize 64
#define llgoFiberFrameRAOff 0
#define llgoFiberFrameArgOff 52

/* fiberGuardMargin: the live sp must leave this much room above stackLo.
 * 64 bytes hold the swap frame, 4 bytes hold the canary word: requiring
 * sp >= stackLo+68 means the frame base (sp-64) lands at or above
 * stackLo+4, so no swap-frame slot can clobber the canary, and the saved
 * from->sp stays strictly above stackLo, keeping the parked stack inside
 * the collector's ctx.stackLo < ctx.sp scan window. Do not shave this to
 * 64: sp == stackLo+64 parks the frame base exactly on the floor, where
 * the ra slot silently overwrites the canary. */
#define llgoFiberGuardMargin (llgoFiberFrameSize + 4)

_Static_assert(llgoFiberGuardMargin == 68, "swap guard margin must stay 64-byte frame + 4-byte canary");
_Static_assert(llgoFiberFrameSize == 64, "swap frame must stay 64 bytes");
_Static_assert(llgoFiberFrameRAOff == 0, "ra slot must stay at the swap frame base");
_Static_assert(llgoFiberFrameArgOff == 52, "a0 argument slot must stay inside the swap frame");

/* The device file (targets/device/esp) provides _write and the linked libc
 * archive provides exit and stdio; they are declared directly so this wrapper
 * stays freestanding like its sibling riscv32 files. All ilp32 slots are 32
 * bits, so the return/size type mismatches with newlib's prototypes are
 * harmless and the results are ignored. */
extern int _write(int fd, const void *buf, unsigned int len);
extern int fflush(void *streams);
/* Weak: SRAM-direct links carry no libc stdio globals at all, and the fatal
 * path must stay linkable there. IDF firmware resolves it to the real FILE*. */
extern void *stdout __attribute__((weak)); /* the libc's global FILE* (the actual type is opaque
                         here; this file stays freestanding) */
extern void exit(int status) __attribute__((noreturn));

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
    /* A naked function takes basic asm only, so the canary constant reaches
     * the assembler as a stringized literal instead of an asm operand. */
    __asm__ volatile(
        // Stack-overflow guard. baremetalFiberContext lays out sp at offset
        // 0, stackLo at 4, stackHi at 8 (proc_fiber_baremetal.go). A context
        // with stackLo == 0 has no configured bounds; skip the guard for it.
        "lw t0, 4(a0)\n"
        "beqz t0, 1f\n"
        // Canary: the floor word of the outgoing fiber's stack must hold the
        // magic. A mismatch means a frame wrote at or below the floor since
        // the last switch, even if sp has since recovered.
        "lw t1, 0(t0)\n"
        // A zero floor word means the canary was never planted (or the
        // platform reused the word): skip the canary test and rely on the sp
        // bound below, which catches a live overflow without owning a
        // dedicated slot in the stack.
        "beqz t1, 3f\n"
        "li t2, " llgoFiberStr(llgoFiberStackCanary) "\n"
        "bne t1, t2, 2f\n"
        "3:\n"
        // SP bound: the 64-byte swap frame below must stay clear of the
        // canary word (see fiberGuardMargin): frame base = sp-64 must be
        // >= stackLo+4, i.e. sp >= stackLo+68.
        "addi t0, t0, " llgoFiberStr(llgoFiberGuardMargin) "\n"
        "bltu sp, t0, 2f\n"
        "1:\n"
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
        "ret\n"
        // Overflow: the outgoing fiber's stack is gone, so run the handler on
        // the target context's saved stack. The G -> scheduler swaps - the
        // only direction that can overflow - land on the 16 KiB bootstrap
        // stack, far above the handler's small frame. The scheduler -> G
        // direction would land on the target G's parked stack, which is only
        // reachable if the bootstrap stack itself overflowed first; that
        // fallback is accepted because the scheduler loop is flat code with
        // no recursion, so its own stack staying healthy is a build-in
        // assumption, not a runtime guarantee.
        "2:\n"
        "lw sp, 0(a1)\n"
        "call llgoFiberStackOverflow\n");
}

/* Terminal path of the swap guard: report the overflow through the same
 * console the runtime's print path drains (the device file's _write) and exit
 * through newlib's semihosting exit, which esp-emu and QEMU intercept. There
 * is no traceback: emu images link no function-info tables, and resuming a
 * fiber whose stack already crossed its floor cannot unwind reliably.
 *
 * fflush(0) drains newlib's stdio buffers first: println output sits in a
 * full buffer on these targets (the emulated console is not a tty), so an
 * unbuffered _write without the flush would land the fatal line ahead of
 * already-printed program output. */
void llgoFiberStackOverflow(unsigned int *from)
{
    static const char msg[] = "fatal error: goroutine stack overflow\n";
    long sp;
    __asm__ volatile("mv %0, sp" : "=r"(sp));
    /* from arrives in a0 from the swap's guard-fail branch: [0]=parked sp,
     * [4]=stackLo. The floor word holds the canary unless something wrote
     * over it - its actual value names the culprit. */
    long stackLo = (long)from[1];
    long floor = *(long *)stackLo;
    char buf[220];
    int n = 0;
    const char hex[] = "0123456789abcdef";
    buf[n++] = '['; buf[n++] = 'c'; buf[n++] = ']'; buf[n++] = ' ';
    const char *labels[6] = {"from=", " w0=", " sp=", " lo=", " hi=", " can="};
    long vals[6] = {(long)from, (long)from[0], sp, stackLo, (long)from[2], floor};
    for (int k = 0; k < 6; k++) {
        const char *l = labels[k];
        while (*l)
            buf[n++] = *l++;
        long v = vals[k];
        for (int i = 7; i >= 0; i--)
            buf[n++] = hex[(v >> (i * 4)) & 0xf];
        buf[n++] = ' ';
    }
    buf[n++] = '\n';
    if (stdout)
        fflush(stdout);
    _write(2, buf, n);
    /* fflush(stdout), not fflush(0): flushing every stream walks the full
     * FILE list and faults on these targets (measured on ESP-IDF, where the
     * list carries per-task TLS streams). */
    if (stdout)
        fflush(stdout);
    _write(2, msg, sizeof msg - 1);
    exit(2);
}
