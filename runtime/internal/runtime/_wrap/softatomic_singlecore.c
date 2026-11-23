#include <stdbool.h>
#include <stdint.h>

// These targets run on one core. Serialize the software atomics with
// interrupt handlers and use full barriers for every memory-order request.
static unsigned atomic_enter(void) {
    unsigned state;
#if defined(__riscv)
    __asm__ volatile(".option push\n\t"
                     ".option arch, +zicsr\n\t"
                     "csrrci %0, mstatus, 8\n\t"
                     ".option pop\n\t"
                     "fence rw, rw"
                     : "=r"(state) :: "memory");
#else
    __asm__ volatile("mrs %0, primask\n\t"
                     "cpsid i\n\t"
                     "dmb"
                     : "=r"(state) :: "memory");
#endif
    return state;
}

static void atomic_leave(unsigned state) {
#if defined(__riscv)
    __asm__ volatile("fence rw, rw" ::: "memory");
    if (state & 8) {
        __asm__ volatile(".option push\n\t"
                         ".option arch, +zicsr\n\t"
                         "csrsi mstatus, 8\n\t"
                         ".option pop" ::: "memory");
    }
#else
    __asm__ volatile("dmb\n\tmsr primask, %0" :: "r"(state) : "memory");
#endif
}

#define ATOMIC_FETCH(T, N, NAME, OP) \
    T atomic_##NAME##N(volatile T *p, T value, int order) \
        __asm__("__atomic_fetch_" #NAME "_" #N); \
    T atomic_##NAME##N(volatile T *p, T value, int order) { \
        (void)order; \
        unsigned state = atomic_enter(); \
        T old = *p; \
        *p = old OP value; \
        atomic_leave(state); \
        return old; \
    }

#define ATOMIC_WIDTH(T, N) \
    T atomic_load##N(const volatile T *p, int order) \
        __asm__("__atomic_load_" #N); \
    T atomic_load##N(const volatile T *p, int order) { \
        (void)order; \
        unsigned state = atomic_enter(); \
        T value = *p; \
        atomic_leave(state); \
        return value; \
    } \
    void atomic_store##N(volatile T *p, T value, int order) \
        __asm__("__atomic_store_" #N); \
    void atomic_store##N(volatile T *p, T value, int order) { \
        (void)order; \
        unsigned state = atomic_enter(); \
        *p = value; \
        atomic_leave(state); \
    } \
    T atomic_exchange##N(volatile T *p, T value, int order) \
        __asm__("__atomic_exchange_" #N); \
    T atomic_exchange##N(volatile T *p, T value, int order) { \
        (void)order; \
        unsigned state = atomic_enter(); \
        T old = *p; \
        *p = value; \
        atomic_leave(state); \
        return old; \
    } \
    bool atomic_compare##N(volatile T *p, T *expected, T value, int success, int failure) \
        __asm__("__atomic_compare_exchange_" #N); \
    bool atomic_compare##N(volatile T *p, T *expected, T value, int success, int failure) { \
        (void)success; \
        (void)failure; \
        unsigned state = atomic_enter(); \
        T old = *p; \
        bool equal = old == *expected; \
        if (equal) \
            *p = value; \
        else \
            *expected = old; \
        atomic_leave(state); \
        return equal; \
    } \
    ATOMIC_FETCH(T, N, add, +) \
    ATOMIC_FETCH(T, N, and, &) \
    ATOMIC_FETCH(T, N, or, |)

#if defined(__riscv)
ATOMIC_WIDTH(uint32_t, 4)
#endif
ATOMIC_WIDTH(uint64_t, 8)
