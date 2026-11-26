/* LLGo vendored __atomic_* fallback for riscv32 targets (RV32IMC, no A
 * extension). LLVM lowers every atomic operation on this target to the
 * __atomic_* libcalls; compiler-rt's atomic.c is excluded from the rtlib
 * because its spinlocks deadlock when an interrupt interrupts a holder, so
 * this file provides the libcall set instead.
 *
 * Every RMW path runs with interrupts masked: csrrc returns the previous
 * mstatus with MIE cleared and csrrs writes only the MIE bit back, so a
 * caller that was already running with interrupts masked stays masked and
 * every other caller resumes exactly as it left. With interrupts masked,
 * each guarded sequence is atomic against interrupt handlers on this hart;
 * the ESP32 baremetal runtime is single-hart, so no other mutual exclusion
 * is required. Plain sized loads and stores are single RV32 instructions and
 * need no masking. Non-maskable interrupts are not covered.
 *
 * All definitions are weak: llgo's Go runtime provides strong definitions of
 * the same libcalls, and the strong versions must win when both link.
 */
#include <stddef.h>
#include <stdint.h>

/* Clang objects if you redefine a builtin. Define the size-generalized
 * operations under internal names and let this pragma emit them under the
 * libcall names, the same way compiler-rt's own atomic.c does. */
#pragma redefine_extname atomic_load_c __atomic_load
#pragma redefine_extname atomic_store_c __atomic_store
#pragma redefine_extname atomic_exchange_c __atomic_exchange
#pragma redefine_extname atomic_compare_exchange_c __atomic_compare_exchange
#pragma redefine_extname atomic_clear_c __atomic_clear
#pragma redefine_extname atomic_test_and_set_c __atomic_test_and_set
#pragma redefine_extname atomic_is_lock_free_c __atomic_is_lock_free

#define LLGO_ATOMIC_MIE 0x8u

static inline unsigned int atomic_irq_save(void)
{
    unsigned int old;
    __asm__ volatile ("csrrc %0, mstatus, %1" : "=r"(old) : "r"(LLGO_ATOMIC_MIE) : "memory");
    return old;
}

static inline void atomic_irq_restore(unsigned int saved)
{
    __asm__ volatile ("csrrs zero, mstatus, %0" :: "r"(saved & LLGO_ATOMIC_MIE) : "memory");
}

/* Sized loads and stores for 1/2/4 bytes are single instructions. */
#define LLGO_ATOMIC_LOAD_STORE(N, T)                                            \
    __attribute__((weak)) T __atomic_load_##N(const T *p, int model)            \
    {                                                                           \
        (void)model;                                                            \
        return *p;                                                              \
    }                                                                           \
    __attribute__((weak)) void __atomic_store_##N(T *p, T val, int model)       \
    {                                                                           \
        (void)model;                                                            \
        *p = val;                                                               \
    }

/* 64-bit accesses are two RV32 instructions; mask interrupts so an interrupt
 * handler cannot observe or interleave a torn value. */
__attribute__((weak)) uint64_t __atomic_load_8(const uint64_t *p, int model)
{
    unsigned int saved = atomic_irq_save();
    uint64_t v = *p;
    atomic_irq_restore(saved);
    (void)model;
    return v;
}

__attribute__((weak)) void __atomic_store_8(uint64_t *p, uint64_t val, int model)
{
    unsigned int saved = atomic_irq_save();
    *p = val;
    atomic_irq_restore(saved);
    (void)model;
}

/* RMW: exchange and fetch_{add,sub,and,or,xor,nand}. The fetched value is
 * returned; nand stores ~(old & val). */
#define LLGO_ATOMIC_EXCHANGE(N, T)                                              \
    __attribute__((weak)) T __atomic_exchange_##N(T *p, T val, int model)       \
    {                                                                           \
        unsigned int saved = atomic_irq_save();                                 \
        T old = *p;                                                             \
        *p = val;                                                               \
        atomic_irq_restore(saved);                                              \
        (void)model;                                                            \
        return old;                                                             \
    }

#define LLGO_ATOMIC_FETCH(N, T, NAME, EXPR)                                     \
    __attribute__((weak)) T __atomic_fetch_##NAME##_##N(T *p, T val, int model) \
    {                                                                           \
        unsigned int saved = atomic_irq_save();                                 \
        T old = *p;                                                             \
        *p = (T)(EXPR);                                                         \
        atomic_irq_restore(saved);                                              \
        (void)model;                                                            \
        return old;                                                             \
    }

#define LLGO_ATOMIC_COMPARE_EXCHANGE(N, T)                                      \
    __attribute__((weak)) _Bool __atomic_compare_exchange_##N(                  \
        T *p, T *expected, T desired, int success_model, int failure_model)     \
    {                                                                           \
        unsigned int saved = atomic_irq_save();                                 \
        T cur = *p;                                                             \
        _Bool ok = cur == *expected;                                            \
        if (ok) {                                                               \
            *p = desired;                                                       \
        } else {                                                                \
            *expected = cur;                                                    \
        }                                                                       \
        atomic_irq_restore(saved);                                              \
        (void)success_model;                                                    \
        (void)failure_model;                                                    \
        return ok;                                                              \
    }

#define LLGO_ATOMIC_FETCH_SET(N, T)                                             \
    LLGO_ATOMIC_FETCH(N, T, add, old + val)                                     \
    LLGO_ATOMIC_FETCH(N, T, sub, old - val)                                     \
    LLGO_ATOMIC_FETCH(N, T, and, old & val)                                     \
    LLGO_ATOMIC_FETCH(N, T, or, old | val)                                      \
    LLGO_ATOMIC_FETCH(N, T, xor, old ^ val)                                     \
    LLGO_ATOMIC_FETCH(N, T, nand, ~(old & val))

#define LLGO_ATOMIC_RMW(N, T)                                                   \
    LLGO_ATOMIC_EXCHANGE(N, T)                                                  \
    LLGO_ATOMIC_FETCH_SET(N, T)                                                 \
    LLGO_ATOMIC_COMPARE_EXCHANGE(N, T)

#define LLGO_ATOMIC_WIDTH(N, T)                                                 \
    LLGO_ATOMIC_LOAD_STORE(N, T)                                                \
    LLGO_ATOMIC_RMW(N, T)

LLGO_ATOMIC_WIDTH(1, uint8_t)
LLGO_ATOMIC_WIDTH(2, uint16_t)
LLGO_ATOMIC_WIDTH(4, uint32_t)
LLGO_ATOMIC_RMW(8, uint64_t)

/* Size-generalized forms. Clang's frontend emits these for atomic operations
 * whose size is not a power-of-two word size, passing the size as the first
 * argument; the CAS form takes no weak flag. The obj/expected/desired buffers
 * are caller allocated. */
__attribute__((weak)) void atomic_load_c(unsigned int size, const void *src, void *dst, int model)
{
    unsigned int saved = atomic_irq_save();
    const unsigned char *from = src;
    unsigned char *to = dst;
    for (unsigned int i = 0; i < size; i++) {
        to[i] = from[i];
    }
    atomic_irq_restore(saved);
    (void)model;
}

__attribute__((weak)) void atomic_store_c(unsigned int size, void *dst, const void *src, int model)
{
    unsigned int saved = atomic_irq_save();
    const unsigned char *from = src;
    unsigned char *to = dst;
    for (unsigned int i = 0; i < size; i++) {
        to[i] = from[i];
    }
    atomic_irq_restore(saved);
    (void)model;
}

__attribute__((weak)) void atomic_exchange_c(unsigned int size, void *obj, const void *desired, void *ret, int model)
{
    unsigned int saved = atomic_irq_save();
    unsigned char *cell = obj;
    const unsigned char *in = desired;
    unsigned char *out = ret;
    for (unsigned int i = 0; i < size; i++) {
        unsigned char old = cell[i];
        cell[i] = in[i];
        out[i] = old;
    }
    atomic_irq_restore(saved);
    (void)model;
}

__attribute__((weak)) _Bool atomic_compare_exchange_c(unsigned int size, void *obj, void *expected, const void *desired, int success_model, int failure_model)
{
    unsigned int saved = atomic_irq_save();
    unsigned char *cell = obj;
    unsigned char *expect = expected;
    const unsigned char *want = desired;
    _Bool ok = 1;
    for (unsigned int i = 0; i < size; i++) {
        if (cell[i] != expect[i]) {
            ok = 0;
            break;
        }
    }
    if (ok) {
        for (unsigned int i = 0; i < size; i++) {
            cell[i] = want[i];
        }
    } else {
        for (unsigned int i = 0; i < size; i++) {
            expect[i] = cell[i];
        }
    }
    atomic_irq_restore(saved);
    (void)success_model;
    (void)failure_model;
    return ok;
}

/* GCC's atomic_flag builtins clear a flag byte through this call. */
__attribute__((weak)) void atomic_clear_c(unsigned char *p, int model)
{
    (void)model;
    *p = 0;
}

/* C11 atomic_flag test-and-set and the lock-free query. Go IR never emits
 * these, but C translation units using stdatomic may, and this archive
 * replaces compiler-rt's atomic.c which used to serve them. */
__attribute__((weak)) _Bool atomic_test_and_set_c(void *ptr, int model)
{
    (void)model;
    unsigned int saved = atomic_irq_save();
    _Bool old = *(volatile unsigned char *)ptr != 0;
    *(volatile unsigned char *)ptr = 1;
    atomic_irq_restore(saved);
    return old;
}

__attribute__((weak)) _Bool atomic_is_lock_free_c(size_t size, const void *ptr)
{
    return ((uintptr_t)ptr & (size - 1)) == 0;
}
