#define _GNU_SOURCE
#include <errno.h>
#include <linux/futex.h>
#include <linux/types.h>
#include <stdint.h>
#include <sys/syscall.h>
#include <unistd.h>

#define LLGO_LINUX_MAX_WAIT_NANOS UINT64_C(86400000000000)

#ifdef SYS_futex
#define LLGO_SYS_FUTEX SYS_futex
#else
#define LLGO_SYS_FUTEX SYS_futex_time64
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
    uint64_t sec = timeout_ns / UINT64_C(1000000000);
    uint64_t nsec = timeout_ns % UINT64_C(1000000000);
    int saved_errno = errno;
    int error;

    // The syscall ABI is independent of libc's time_t and struct timespec.
#ifdef SYS_futex_time64
    struct { int64_t sec, nsec; } time64 = { (int64_t)sec, (int64_t)nsec };
    long result = syscall(SYS_futex_time64, addr, FUTEX_WAIT_PRIVATE, value, &time64, 0, 0);
    error = result < 0 ? errno : 0;
#ifdef SYS_futex
    if (error == ENOSYS)
#endif
#endif
#ifdef SYS_futex
    {
        // The one-day cap keeps sec representable even on 32-bit kernels.
        struct { __kernel_long_t sec, nsec; } legacy = { (__kernel_long_t)sec, (__kernel_long_t)nsec };
        long result = syscall(SYS_futex, addr, FUTEX_WAIT_PRIVATE, value, &legacy, 0, 0);
        error = result < 0 ? errno : 0;
    }
#endif
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
