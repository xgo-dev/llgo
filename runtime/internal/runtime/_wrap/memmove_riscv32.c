/* Optimized word-block memmove/memcpy for RV32IMC baremetal (ESP32-C3/S31).
 *
 * The newlib-esp32 RV32 build compiles only its PREFER_SIZE_OVER_SPEED
 * variant, whose memmove and memcpy are plain byte loops (~7 cycles/byte on
 * S31; the copy() benchmark measured 14402 cycles per 1 KiB against 1486 for
 * an optimized newlib). These llgo-prefixed replacements are called from
 * z_slice.go through the Go wrappers in z_memmove_riscv32.go, so slice copy
 * and append — the copy()/append base layer — get word-block speed.
 *
 * The C functions cannot simply take the libc memmove/memcpy names: the libc
 * archives sit before the package archives on the link line, so newlib's
 * lazy symbol entries claim those names first and a same-name definition in
 * this archive would never (or conflict-)link. Unique llgo names give the
 * references a single provider, mirroring the atomic_riscv32 shim.
 *
 * All word accesses stay naturally aligned: RV32 cores may fault on
 * unaligned loads/stores, so a misaligned source is reassembled from the two
 * aligned words spanning it. Reassembly reads up to 3 bytes below the source
 * start and up to 3 bytes past the source end (it never writes outside the
 * destination range) — safe on this target's single contiguous RAM space,
 * and newlib's own RV32 asm memcpy reads spanning words the same way. Every
 * store is a single aligned word (or byte) instruction, preserving the
 * indivisible pointer-sized writes that Go relies on for word-aligned
 * ranges.
 *
 * no_builtin stops LLVM from recognizing these loops as memcpy/memmove and
 * lowering them right back into libc calls — infinite recursion.
 */

#include <stdint.h>
#include <stddef.h>

typedef uint32_t __attribute__((may_alias)) llgo_word;

#define LLGO_NO_BUILTIN __attribute__((no_builtin))

/* Copy n bytes ascending. Overlapping is only safe when dst <= src or the
 * ranges do not overlap; llgoMemmove routes overlapping dst > src copies to
 * llgo_copy_backward instead. */
static LLGO_NO_BUILTIN void *llgo_copy_forward(void *dstp, const void *srcp, size_t n)
{
    uint8_t *dst = (uint8_t *)dstp;
    const uint8_t *src = (const uint8_t *)srcp;

    /* n == 0 must not touch either pointer: memcpy(p, q, 0) is allowed with
     * otherwise-invalid or page-boundary pointers, and append() reaches this
     * helper unguarded with zero-length slices. */
    if (n == 0u) {
        return dstp;
    }

    /* Byte head: advance both pointers in lockstep, so the relative mod-4
     * offset between src and dst never changes, until dst is word aligned.
     * At most three bytes, unrolled: a loop here risks being rewritten into
     * a memcpy call. */
    if (((uintptr_t)dst & 3u) != 0u) {
        if (((uintptr_t)dst & 1u) != 0u) {
            if (n-- == 0u) {
                return dstp;
            }
            *dst++ = *src++;
        }
        if (((uintptr_t)dst & 2u) != 0u) {
            if (n-- == 0u) {
                return dstp;
            }
            *dst++ = *src++;
            if (n-- == 0u) {
                return dstp;
            }
            *dst++ = *src++;
        }
    }

    if (((uintptr_t)src & 3u) == 0u) {
        /* Both pointers word aligned: bulk word blocks. */
        while (n >= 16u) {
            const llgo_word *sw = (const llgo_word *)src;
            llgo_word *dw = (llgo_word *)dst;
            llgo_word w0 = sw[0];
            llgo_word w1 = sw[1];
            llgo_word w2 = sw[2];
            llgo_word w3 = sw[3];
            dw[0] = w0;
            dw[1] = w1;
            dw[2] = w2;
            dw[3] = w3;
            src += 16u;
            dst += 16u;
            n -= 16u;
        }
        while (n >= 4u) {
            *(llgo_word *)dst = *(const llgo_word *)src;
            src += 4u;
            dst += 4u;
            n -= 4u;
        }
    } else {
        /* dst is aligned but src is not: reassemble each destination word
         * from the two aligned source words spanning it, so no access is
         * ever unaligned. sh is 8, 16, or 24 here. */
        uintptr_t sh = ((uintptr_t)src & 3u) * 8u;
        const llgo_word *sw = (const llgo_word *)((uintptr_t)src & ~(uintptr_t)3u);
        llgo_word cur = *sw++;
        while (n >= 4u) {
            llgo_word nxt = *sw++;
            llgo_word out;
            if (sh == 8u) {
                out = (cur >> 8u) | (nxt << 24u);
            } else if (sh == 16u) {
                out = (cur >> 16u) | (nxt << 16u);
            } else {
                out = (cur >> 24u) | (nxt << 8u);
            }
            *(llgo_word *)dst = out;
            src += 4u;
            dst += 4u;
            n -= 4u;
            cur = nxt;
        }
    }

    /* Byte tail, at most three bytes. */
    if ((n & 2u) != 0u) {
        dst[0] = src[0];
        dst[1] = src[1];
        dst += 2u;
        src += 2u;
    }
    if ((n & 1u) != 0u) {
        dst[0] = src[0];
    }
    return dstp;
}

/* Copy n bytes descending. Only llgoMemmove reaches this: it is the
 * direction that keeps dst > src overlapping copies from clobbering their
 * own source. */
static LLGO_NO_BUILTIN void *llgo_copy_backward(void *dstp, const void *srcp, size_t n)
{
    uint8_t *dst = (uint8_t *)dstp + n;
    const uint8_t *src = (const uint8_t *)srcp + n;

    /* Byte tail from the end: step both pointers down in lockstep until dst
     * is word aligned. */
    if (((uintptr_t)dst & 3u) != 0u) {
        if (((uintptr_t)dst & 1u) != 0u) {
            if (n-- == 0u) {
                return dstp;
            }
            *--dst = *--src;
        }
        if (((uintptr_t)dst & 2u) != 0u) {
            if (n-- == 0u) {
                return dstp;
            }
            *--dst = *--src;
            if (n-- == 0u) {
                return dstp;
            }
            *--dst = *--src;
        }
    }

    if (((uintptr_t)src & 3u) == 0u) {
        /* Both end pointers word aligned: bulk word blocks. */
        while (n >= 16u) {
            const llgo_word *sw = (const llgo_word *)src;
            llgo_word *dw = (llgo_word *)dst;
            llgo_word w0, w1, w2, w3;
            sw -= 4;
            dw -= 4;
            w0 = sw[0];
            w1 = sw[1];
            w2 = sw[2];
            w3 = sw[3];
            dw[0] = w0;
            dw[1] = w1;
            dw[2] = w2;
            dw[3] = w3;
            src -= 16u;
            dst -= 16u;
            n -= 16u;
        }
        while (n >= 4u) {
            src -= 4u;
            dst -= 4u;
            *(llgo_word *)dst = *(const llgo_word *)src;
            n -= 4u;
        }
    } else {
        /* dst is aligned (downwards) but src is not: reassemble each
         * destination word from the two aligned source words spanning it.
         * sh is 8, 16, or 24 here. */
        uintptr_t sh = ((uintptr_t)src & 3u) * 8u;
        const llgo_word *sw = (const llgo_word *)((uintptr_t)src & ~(uintptr_t)3u);
        llgo_word cur = *sw;
        while (n >= 4u) {
            llgo_word prev = *--sw;
            llgo_word out;
            if (sh == 8u) {
                out = (prev >> 8u) | (cur << 24u);
            } else if (sh == 16u) {
                out = (prev >> 16u) | (cur << 16u);
            } else {
                out = (prev >> 24u) | (cur << 8u);
            }
            dst -= 4u;
            *(llgo_word *)dst = out;
            src -= 4u;
            n -= 4u;
            cur = prev;
        }
    }

    /* Front byte tail, at most three bytes. */
    if ((n & 2u) != 0u) {
        *--dst = *--src;
        *--dst = *--src;
    }
    if ((n & 1u) != 0u) {
        *--dst = *--src;
    }
    return dstp;
}

LLGO_NO_BUILTIN void *llgoMemmove(void *dst, const void *src, size_t n)
{
    if (n == 0u || dst == src) {
        return dst;
    }
    if ((uintptr_t)dst > (uintptr_t)src &&
        (uintptr_t)dst - (uintptr_t)src < n) {
        return llgo_copy_backward(dst, src, n);
    }
    return llgo_copy_forward(dst, src, n);
}

/* llgoMemcpy follows the forward path: overlapping inputs are undefined, so
 * the direction check would only add a branch to every append. */
LLGO_NO_BUILTIN void *llgoMemcpy(void *dst, const void *src, size_t n)
{
    return llgo_copy_forward(dst, src, n);
}
