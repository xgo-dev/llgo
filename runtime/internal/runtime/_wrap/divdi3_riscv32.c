/* LLGo vendored 64-bit integer division/modulo for riscv32 targets
 * (RV32IMC: no 64-bit divider; 32-bit divu/remu is hardware).
 *
 * LLVM lowers every Go int64/uint64 division and remainder on this target
 * to the __divdi3/__moddi3/__udivdi3/__umoddi3 libcalls, which compiler-rt
 * serves with a shift-subtract loop running once per quotient bit, each
 * iteration a full 64-bit shift/compare/subtract (~1000 cycles per division
 * on large operands). GCC/libgcc serves the same libcall set with Knuth
 * Algorithm D driven by the hardware 32-bit divider (~90 cycles). This file
 * overrides the libcall set the libgcc way:
 *
 *   - divisor high half == 0 and dividend high half == 0: one hardware
 *     divu/remu pair. This keeps the small-operand fast path that programs
 *     like the pi1000 spigot bench live on.
 *   - divisor high half == 0 (64/32): Knuth base-2^16 quotient digits, each
 *     estimated with one hardware 32/32 division. 32-bit arithmetic only.
 *   - divisor full 64 bits: Knuth Algorithm D at base 2^16 over half-digit
 *     arrays (Hacker's Delight fig. 10-1), two hardware 32/32 divisions.
 *
 * The C `/` and `%` operators on 32-bit operands compile to the hardware
 * divu/remu of the M extension; there is no recursion into this file.
 *
 * All definitions are weak, mirroring the __atomic_* shim: the libcall ABI
 * is shared with compiler-rt's builtins, so the two may coexist in one link
 * (llgo's runtime object wins normal Go links) without duplicate-symbol
 * errors.
 *
 * Division by zero is unspecified at the libcall level (Go checks the
 * divisor in generated code before the call). The small-operand path
 * reaches the hardware divu, whose convention is quotient all ones,
 * remainder = dividend; the wide paths terminate with meaningless values.
 */
#include <stdint.h>

typedef uint32_t u32;
typedef int32_t i32;
typedef uint64_t u64;
typedef int64_t i64;

/* Count leading zeros without the Zbb clz instruction (RV32IMC). De Bruijn
 * multiplication, same table method libgcc uses on this target. */
static inline __attribute__((always_inline)) u32 llgo_clz32(u32 x)
{
    static const unsigned char tab[32] = {
        31, 22, 30, 21, 18, 10, 29, 2, 20, 17, 15, 13, 9, 6, 28, 1,
        23, 19, 11, 3, 16, 14, 7, 24, 12, 4, 8, 25, 5, 26, 27, 0,
    };
    x |= x >> 1;
    x |= x >> 2;
    x |= x >> 4;
    x |= x >> 8;
    x |= x >> 16;
    return (u32)tab[(x * 0x07C4ACDDu) >> 27];
}

/* Next Knuth base-2^16 quotient digit for dividing by the normalized
 * two-digit divisor (v1:v0). win holds the top two digits of the current
 * three-digit window (win < v1*2^16 + v0), next is the third digit. Returns
 * the digit (< 2^16), leaving the window remainder to the caller.
 *
 * The estimate win/v1 is high by at most 2; the correction loop mirrors
 * Knuth's: while the estimate overflows a digit or its trial product
 * exceeds the trial remainder, decrement (at most twice). rhat can only
 * exceed 2^16 in the iteration that proves the estimate exact, and the test
 * keeps rhat < 2^16 so every product fits 32 bits. */
static inline __attribute__((always_inline)) u32 llgo_digit(u32 win, u32 v1, u32 v0, u32 next)
{
    u32 q = win / v1;
    u32 r = win - q * v1; /* == win % v1, cheaper than a second divider op */
    for (;;) {
        if (q < 0x10000u && q * v0 <= (r << 16) + next) {
            return q;
        }
        q--;
        r += v1;
        if (r >= 0x10000u) {
            return q;
        }
    }
}

/* Divide the 64-bit value (u1:u0) by v, where v is normalized (top bit set)
 * and u1 < v, so the quotient fits 32 bits. Returns the quotient and stores
 * the remainder (still normalized) in *rp. Two base-2^16 quotient digits.
 *
 * The window remainder is a value below v computed with 32-bit modular
 * arithmetic: after Knuth's correction the true remainder fits 32 bits, so
 * wrapping intermediates cancel. */
static inline __attribute__((always_inline)) u32 llgo_divnorm(u32 u1, u32 u0, u32 v, u32 *rp)
{
    u32 v1 = v >> 16;
    u32 v0 = v & 0xFFFFu;
    u32 q1 = llgo_digit(u1, v1, v0, u0 >> 16);
    u32 rho = (((u1 - q1 * v1) << 16) | (u0 >> 16)) - q1 * v0;
    u32 q2 = llgo_digit(rho, v1, v0, u0 & 0xFFFFu);
    *rp = (((rho << 16) | (u0 & 0xFFFFu)) - q2 * v0 - (q2 * v1 << 16));
    return (q1 << 16) | q2;
}

/* Divide the 64-bit dividend n<<s (carrying overflow bits e above bit 63)
 * by the normalized divisor d<<s >= 2^32, at Knuth base 2^16: the dividend
 * spans at most six significant half-digits over the four-digit divisor,
 * with a zero pad digit on top (the canonical Algorithm D form, which keeps
 * every estimate window's leading digit below the divisor's leading digit
 * and thus every estimate within two corrections of the true digit). The
 * loop follows Hacker's Delight fig. 10-1: estimate from the window's top
 * digits against the divisor's leading digit, correct, multiply-subtract
 * across all divisor digits with a signed borrow, and add back in the rare
 * case the top borrow went negative. Returns the 32-bit quotient (the
 * quotient of a 64-bit dividend by a divisor >= 2^32 always fits 32 bits,
 * so the leading digit steps yield zero) and stores the un-shifted
 * remainder in *rp. */
static u32 llgo_divbig(u32 e, u64 nsh, u64 dv, u32 s, u64 *rp)
{
    u32 m[7]; /* half-digits, little-endian; m[6] is the zero pad */
    u32 d[4];
    u32 q[3] = {0, 0, 0};
    u32 k;
    int j, i;
    m[0] = nsh & 0xFFFFu;
    m[1] = (u32)(nsh >> 16) & 0xFFFFu;
    m[2] = (u32)(nsh >> 32) & 0xFFFFu;
    m[3] = (u32)(nsh >> 48) & 0xFFFFu;
    m[4] = e & 0xFFFFu;
    m[5] = e >> 16;
    m[6] = 0;
    d[0] = dv & 0xFFFFu;
    d[1] = (u32)(dv >> 16) & 0xFFFFu;
    d[2] = (u32)(dv >> 32) & 0xFFFFu;
    d[3] = (u32)(dv >> 48) & 0xFFFFu;
    k = d[3];
    /* Quotient digit steps, Hacker's Delight fig. 10-1 indexing: step j
     * estimates from the window (m[j+4], m[j+3]) whose leading digit is the
     * dividend's pad for the top step, which keeps the estimate within two
     * corrections of the true digit. A dividend whose top significant digit
     * sits below the divisor's top (m[3] == 0) is smaller than the divisor
     * and needs no steps at all. */
    if (m[5]) {
        j = 2;
    } else if (m[4]) {
        j = 1;
    } else if (m[3]) {
        j = 0;
    } else {
        /* Smaller than the divisor: quotient zero, remainder the dividend. */
        *rp = (((u64)e << 32) | nsh) >> s;
        return 0;
    }
    for (; j >= 0; j--) {
        u32 w = (m[j + 4] << 16) | m[j + 3];
        u32 qhat = w / k;
        u32 r2 = w % k;
        while (qhat >= 0x10000u || qhat * d[2] > r2 * 0x10000u + m[j + 2]) {
            qhat--;
            r2 += k;
            if (r2 >= 0x10000u) {
                break;
            }
        }
        i32 borrow = 0;
        for (i = 0; i < 4; i++) {
            u32 p = qhat * d[i];
            i32 t = (i32)m[i + j] - borrow - (i32)(p & 0xFFFFu);
            m[i + j] = (u32)t & 0xFFFFu;
            borrow = (i32)(p >> 16) - (t >> 16);
        }
        i32 t = (i32)m[j + 4] - borrow;
        m[j + 4] = (u32)t & 0xFFFFu;
        if (t < 0) {
            /* Estimate was one too high: add the divisor back. */
            u32 carry = 0;
            qhat--;
            for (i = 0; i < 4; i++) {
                u32 t2 = m[i + j] + d[i] + carry;
                m[i + j] = t2 & 0xFFFFu;
                carry = t2 >> 16;
            }
            m[j + 4] = (m[j + 4] + carry) & 0xFFFFu;
        }
        q[j] = qhat;
    }
    u64 rem = ((u64)m[3] << 48) | ((u64)m[2] << 32) | ((u64)m[1] << 16) | m[0];
    *rp = rem >> s;
    return (u32)(((u64)q[2] << 32) | ((u64)q[1] << 16) | q[0]);
}

/* Unsigned 64/64 division with remainder.
 *
 * Values move through half-word unions: on the ilp32 ABI a 64-bit argument
 * already arrives as a register pair, so splitting by union compiles to
 * nothing, and the hot paths below stay free of 64-bit shift operations —
 * which matter here, because at -Oz every structurally-visible 64-bit
 * shift becomes a call to the __lshrdi3/__ashldi3 helpers rather than an
 * inline instruction pair.
 *
 * The hot small-operand path carries no preamble beyond the two dispatch
 * comparisons, which keeps programs dominated by small divisions (like the
 * pi1000 spigot) at compiler-rt's speed. A zero divisor reaches the
 * hardware divu on the small path (quotient all ones, remainder the
 * dividend — the RISC-V convention; division by zero is unspecified at the
 * libcall level anyway, and Go checks before calling) or terminates with
 * meaningless values on the wide paths. A unit divisor on the wide paths
 * takes the digit steps. */
typedef union {
    u64 v;
    struct {
        u32 lo, hi; /* little-endian: matches the ilp32 register order */
    } h;
} llgo_split64;

static inline __attribute__((always_inline)) u64 llgo_udivmod64(u64 n, u64 d, u64 *rp)
{
    llgo_split64 un, ud, out;
    un.v = n;
    ud.v = d;
    if (ud.h.hi == 0) {
        u32 dv = ud.h.lo;
        if (un.h.hi == 0) {
            /* Both halves fit 32 bits: one hardware pair. */
            u32 q = un.h.lo / dv;
            if (rp) {
                out.h.lo = un.h.lo - q * dv; /* == % */
                out.h.hi = 0;
                *rp = out.v;
            }
            out.h.lo = q;
            out.h.hi = 0;
            return out.v;
        }
        if (dv == 1) {
            if (rp) {
                *rp = 0;
            }
            return n;
        }
        /* 64/32: one hardware division for the top quotient word, then a
         * normalized 32-bit-digit step for the bottom word. Shifting the
         * residual (r1:n_lo) left by the normalization amount s keeps it
         * below v*2^32, so the step's quotient fits 32 bits. */
        u32 q1 = un.h.hi / dv;
        u32 r1 = un.h.hi - q1 * dv; /* == % */
        u32 s = llgo_clz32(dv);
        u32 v = dv << s;
        u32 u1 = (r1 << s) | (s ? un.h.lo >> (32 - s) : 0);
        u32 r2;
        u32 q2 = llgo_divnorm(u1, un.h.lo << s, v, &r2);
        if (rp) {
            /* r2 < v <= 2^32-1, so the shifted remainder fits 32 bits. */
            out.h.lo = r2 >> s;
            out.h.hi = 0;
            *rp = out.v;
        }
        out.h.lo = q2;
        out.h.hi = q1;
        return out.v;
    }
    /* 64/64: align the divisor's top bit and run Algorithm D. The bits
     * shifted out of the dividend (its top s bits, all inside the high
     * half) become its leading digits. */
    u32 s = llgo_clz32(ud.h.hi);
    u32 e = s ? un.h.hi >> (32 - s) : 0; /* dividend's top s bits */
    u64 rbig;
    u32 q = llgo_divbig(e,
                        ((u64)((s ? un.h.hi << s | un.h.lo >> (32 - s) : un.h.hi)) << 32)
                            | un.h.lo << s,
                        ((u64)(ud.h.hi << s | (s ? ud.h.lo >> (32 - s) : 0)) << 32)
                            | ud.h.lo << s,
                        s, &rbig);
    if (rp) {
        *rp = rbig;
    }
    out.h.lo = q;
    out.h.hi = 0;
    return out.v;
}

__attribute__((weak)) u64 __udivdi3(u64 a, u64 b)
{
    return llgo_udivmod64(a, b, 0);
}

__attribute__((weak)) u64 __umoddi3(u64 a, u64 b)
{
    u64 r;
    llgo_udivmod64(a, b, &r);
    return r;
}

__attribute__((weak)) i64 __divdi3(i64 a, i64 b)
{
    /* Branchless sign handling, compiler-rt style: absolute values by xor
     * with the sign mask, quotient sign re-applied with wrap-around. The
     * wrap gives INT64_MIN/-1 = INT64_MIN, matching Go (and hardware)
     * semantics. */
    u64 sa = (u64)((i64)a >> 63);
    u64 sb = (u64)((i64)b >> 63);
    u64 ua = ((u64)a ^ sa) - sa;
    u64 ub = ((u64)b ^ sb) - sb;
    u64 q = llgo_udivmod64(ua, ub, 0);
    q = (q ^ (sa ^ sb)) - (sa ^ sb);
    return (i64)q;
}

/* Combined divide+remainder forms, served for completeness: clang's C
 * front end on riscv32 emits these DIVREM libcalls, and divmoddi4.o also
 * references __udivmoddi4 internally, so the rtlib replacement must cover
 * them when compiler-rt's own copies are excluded. */
__attribute__((weak)) u64 __udivmoddi4(u64 a, u64 b, u64 *rem)
{
    u64 r;
    u64 q = llgo_udivmod64(a, b, &r);
    if (rem) {
        *rem = r;
    }
    return q;
}

__attribute__((weak)) i64 __divmoddi4(i64 a, i64 b, i64 *rem)
{
    u64 sa = (u64)((i64)a >> 63);
    u64 sb = (u64)((i64)b >> 63);
    u64 ua = ((u64)a ^ sa) - sa;
    u64 ub = ((u64)b ^ sb) - sb;
    u64 r;
    u64 q = llgo_udivmod64(ua, ub, &r);
    q = (q ^ (sa ^ sb)) - (sa ^ sb);
    r = (r ^ sa) - sa;
    if (rem) {
        *rem = (i64)r;
    }
    return (i64)q;
}

__attribute__((weak)) i64 __moddi3(i64 a, i64 b)
{
    /* C/Go remainder takes the dividend's sign. */
    u64 sa = (u64)((i64)a >> 63);
    u64 sb = (u64)((i64)b >> 63);
    u64 ua = ((u64)a ^ sa) - sa;
    u64 ub = ((u64)b ^ sb) - sb;
    u64 r;
    llgo_udivmod64(ua, ub, &r);
    r = (r ^ sa) - sa;
    return (i64)r;
}
