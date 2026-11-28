#define _GNU_SOURCE
#include <errno.h>
#include <linux/futex.h>
#include <stdint.h>
#include <sys/syscall.h>
#include <time.h>
#include <unistd.h>

#define LLGO_LINUX_MAX_WAIT_NANOS UINT64_C(86400000000000)

#if defined(SYS_futex_time64) && !defined(SYS_futex)
#define LLGO_SYS_FUTEX SYS_futex_time64
#elif defined(SYS_futex_time64) && defined(_TIME_BITS) && _TIME_BITS == 64
#define LLGO_SYS_FUTEX SYS_futex_time64
#else
#define LLGO_SYS_FUTEX SYS_futex
#endif

int llgo_linux_wait_uint32(uint32_t *addr, uint32_t value)
{
    int saved_errno = errno;
    long result = syscall(LLGO_SYS_FUTEX, addr, FUTEX_WAIT_PRIVATE, value, 0, 0, 0);
    int error = result < 0 ? errno : 0;
    errno = saved_errno;
    return error;
}

int llgo_linux_wait_uint32_timeout(uint32_t *addr, uint32_t value, uint64_t timeout_ns)
{
    if (timeout_ns == 0)
        return ETIMEDOUT;
    if (timeout_ns > LLGO_LINUX_MAX_WAIT_NANOS)
        timeout_ns = LLGO_LINUX_MAX_WAIT_NANOS;
    struct timespec ts;
    ts.tv_sec = (time_t)(timeout_ns / 1000000000ULL);
    ts.tv_nsec = (long)(timeout_ns % 1000000000ULL);
    int saved_errno = errno;
    long result = syscall(LLGO_SYS_FUTEX, addr, FUTEX_WAIT_PRIVATE, value, &ts, 0, 0);
    int error = result < 0 ? errno : 0;
    errno = saved_errno;
    return error;
}

void llgo_linux_wake_uint32(uint32_t *addr)
{
    int saved_errno = errno;
    syscall(LLGO_SYS_FUTEX, addr, FUTEX_WAKE_PRIVATE, 1, 0, 0, 0);
    errno = saved_errno;
}

int llgo_linux_wait_errno_retryable(int err)
{
    return err == 0 || err == EINTR || err == EAGAIN || err == ETIMEDOUT;
}
