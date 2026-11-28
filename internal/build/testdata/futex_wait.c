#define _GNU_SOURCE
#include <assert.h>
#include <errno.h>
#include <stdint.h>
#include <sys/syscall.h>
#include <time.h>
#include <unistd.h>

static long last_syscall;
#if defined(SYS_futex_time64) && defined(SYS_futex)
static int force_time64_missing;
static int rejected_time64_calls;
#endif

static long test_futex(long number, uint32_t *addr, int op, uint32_t value,
                       const void *timeout, int addr2, int value3)
{
    last_syscall = number;
#if defined(SYS_futex_time64) && defined(SYS_futex)
    if (number == SYS_futex_time64 && force_time64_missing) {
        ++rejected_time64_calls;
        errno = ENOSYS;
        return -1;
    }
#endif
    return syscall(number, addr, op, value, timeout, addr2, value3);
}

#define syscall test_futex
#include "../../../runtime/internal/sync/_wrap/sync_linux.c"
#undef syscall

static int64_t monotonic_nanos(void)
{
    struct timespec ts;
    assert(clock_gettime(CLOCK_MONOTONIC, &ts) == 0);
    return (int64_t)ts.tv_sec * INT64_C(1000000000) + ts.tv_nsec;
}

static void check_timeout(void)
{
    uint32_t word = 0;
    const uint64_t timeout = UINT64_C(20000000);
    int64_t start = monotonic_nanos();
    errno = EDOM;
    assert(llgo_linux_wait_uint32_timeout(&word, 0, timeout) == ETIMEDOUT);
    assert(errno == EDOM);
    // A 32-bit legacy futex reading libc's time64 timespec sees zero here.
    assert(monotonic_nanos() - start >= (int64_t)timeout);
}

int main(void)
{
    check_timeout();
#if defined(SYS_futex_time64) && defined(SYS_futex)
    // Simulate a kernel that implements only the legacy syscall.
    force_time64_missing = 1;
    check_timeout();
    assert(rejected_time64_calls == 1);
    assert(last_syscall == SYS_futex);

    uint32_t word = 1;
    assert(llgo_linux_wait_uint32(&word, 0) == EAGAIN);
    assert(last_syscall == SYS_futex);
    llgo_linux_wake_uint32(&word);
    assert(last_syscall == SYS_futex);
    assert(rejected_time64_calls == 1);
#endif
    return 0;
}
