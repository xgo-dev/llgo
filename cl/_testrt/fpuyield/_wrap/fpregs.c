// fpregs.c — FP-register yield probe for _testrt/fpuyield.
//
// On F-extension RISC-V targets (__riscv_flen >= 32) the fill/check pair
// plants a distinct bit pattern in every callee-saved f register plus fcsr
// and verifies it survives a cooperative fiber swap. The bodies are naked
// basic-asm on purpose: compiler-inserted save/restore must not mask a swap
// bug. llgo_fp_fill deliberately violates the callee-saved contract — it is
// the fixture's injected fault, and the Go side only calls it while no FP
// value of its own is live (see in.go). On every other target (soft-float
// rv32, hosted) the pair degrades to no-ops so the fixture's golden output
// stays identical everywhere while hard-float targets keep the
// register-level assertion.
#if defined(__riscv) && defined(__riscv_flen) && __riscv_flen >= 32

/* fs0-fs11 = 0x3f800000+i (i = 0..11); fcsr = 0x1f (frm=RNE, all fflags set).
 * Any nonzero fflags is a legal CSR value, so the write cannot trap and the
 * later frcsr must return it bit-exact. */
__attribute__((naked)) void llgo_fp_fill(void)
{
    __asm__ volatile(
        "li t0, 0x3f800000\n"
        "fmv.w.x fs0, t0\n"
        "li t0, 0x3f800001\n"
        "fmv.w.x fs1, t0\n"
        "li t0, 0x3f800002\n"
        "fmv.w.x fs2, t0\n"
        "li t0, 0x3f800003\n"
        "fmv.w.x fs3, t0\n"
        "li t0, 0x3f800004\n"
        "fmv.w.x fs4, t0\n"
        "li t0, 0x3f800005\n"
        "fmv.w.x fs5, t0\n"
        "li t0, 0x3f800006\n"
        "fmv.w.x fs6, t0\n"
        "li t0, 0x3f800007\n"
        "fmv.w.x fs7, t0\n"
        "li t0, 0x3f800008\n"
        "fmv.w.x fs8, t0\n"
        "li t0, 0x3f800009\n"
        "fmv.w.x fs9, t0\n"
        "li t0, 0x3f80000a\n"
        "fmv.w.x fs10, t0\n"
        "li t0, 0x3f80000b\n"
        "fmv.w.x fs11, t0\n"
        "li t0, 0x1f\n"
        "fscsr t0\n"
        "ret\n");
}

/* Returns 1 when every planted pattern survived the swap, 0 otherwise. */
__attribute__((naked)) int llgo_fp_check(void)
{
    __asm__ volatile(
        "frcsr t0\n"
        "li t1, 0x1f\n"
        "bne t0, t1, 1f\n"
        "li t1, 0x3f800000\n"
        "fmv.x.w t0, fs0\n"
        "bne t0, t1, 1f\n"
        "li t1, 0x3f800001\n"
        "fmv.x.w t0, fs1\n"
        "bne t0, t1, 1f\n"
        "li t1, 0x3f800002\n"
        "fmv.x.w t0, fs2\n"
        "bne t0, t1, 1f\n"
        "li t1, 0x3f800003\n"
        "fmv.x.w t0, fs3\n"
        "bne t0, t1, 1f\n"
        "li t1, 0x3f800004\n"
        "fmv.x.w t0, fs4\n"
        "bne t0, t1, 1f\n"
        "li t1, 0x3f800005\n"
        "fmv.x.w t0, fs5\n"
        "bne t0, t1, 1f\n"
        "li t1, 0x3f800006\n"
        "fmv.x.w t0, fs6\n"
        "bne t0, t1, 1f\n"
        "li t1, 0x3f800007\n"
        "fmv.x.w t0, fs7\n"
        "bne t0, t1, 1f\n"
        "li t1, 0x3f800008\n"
        "fmv.x.w t0, fs8\n"
        "bne t0, t1, 1f\n"
        "li t1, 0x3f800009\n"
        "fmv.x.w t0, fs9\n"
        "bne t0, t1, 1f\n"
        "li t1, 0x3f80000a\n"
        "fmv.x.w t0, fs10\n"
        "bne t0, t1, 1f\n"
        "li t1, 0x3f80000b\n"
        "fmv.x.w t0, fs11\n"
        "bne t0, t1, 1f\n"
        "li a0, 1\n"
        "ret\n"
        "1:\n"
        "li a0, 0\n"
        "ret\n");
}

#else /* soft-float rv32 or hosted: no f register file to probe */

void llgo_fp_fill(void)
{
}

int llgo_fp_check(void)
{
    return 1;
}

#endif
