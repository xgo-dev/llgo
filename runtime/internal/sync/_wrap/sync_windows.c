/* Native Windows synchronization for the hosted LLGo runtime.
 *
 * Keep this file independent of the Windows SDK headers. The declarations
 * below are the complete Kernel32 surface used here; clang still applies the
 * target's MSVC ABI and emits ordinary Kernel32 imports.
 */
typedef unsigned long llgo_dword;
typedef long long llgo_time_t;
typedef int llgo_bool;
typedef void *llgo_srwlock;
typedef void *llgo_condition_variable;
typedef void *llgo_init_once;

#if defined(_WIN64)
typedef unsigned long long llgo_size_t;
#else
typedef unsigned int llgo_size_t;
#endif

#if defined(_WIN64)
#define LLGO_WINAPI
#else
#define LLGO_WINAPI __attribute__((stdcall))
#endif

__declspec(dllimport) llgo_bool LLGO_WINAPI SleepConditionVariableSRW(
    llgo_condition_variable *condition, llgo_srwlock *lock,
    llgo_dword milliseconds, llgo_dword flags);
__declspec(dllimport) llgo_bool LLGO_WINAPI WaitOnAddress(
    volatile void *address, void *compare_address,
    llgo_size_t address_size, llgo_dword milliseconds);

typedef struct {
    llgo_dword low;
    llgo_dword high;
} llgo_filetime;

__declspec(dllimport) void LLGO_WINAPI
GetSystemTimeAsFileTime(llgo_filetime *time);

typedef llgo_bool(LLGO_WINAPI *llgo_init_once_fn)(
    llgo_init_once *once, void *parameter, void **context);
__declspec(dllimport) llgo_bool LLGO_WINAPI InitOnceExecuteOnce(
    llgo_init_once *once, llgo_init_once_fn callback, void *parameter,
    void **context);
__declspec(dllimport) llgo_dword LLGO_WINAPI GetLastError(void);

#define LLGO_INFINITE ((llgo_dword)0xffffffffUL)

enum {
    llgo_error_invalid_parameter = 22,
    llgo_error_timeout = 1460,
    llgo_timedout = 110,
};

typedef void (*llgo_once_fn)(void);

typedef struct {
    llgo_once_fn fn;
} llgo_once_call;

static llgo_bool LLGO_WINAPI llgo_once_callback(
    llgo_init_once *once, void *parameter, void **context)
{
    llgo_once_call *call = (llgo_once_call *)parameter;
    (void)once;
    (void)context;
    call->fn();
    return 1;
}

int llgo_win_once(llgo_init_once *once, llgo_once_fn fn)
{
    llgo_once_call call;
    if (fn == 0)
        return 87; /* ERROR_INVALID_PARAMETER */
    call.fn = fn;
    if (InitOnceExecuteOnce(once, llgo_once_callback, &call, 0))
        return 0;
    return (int)GetLastError();
}

typedef struct {
    llgo_condition_variable *condition;
    llgo_srwlock *lock;
    llgo_dword milliseconds;
    int error;
} llgo_cond_wait_call;

static void *llgo_cond_wait_inactive(void *raw)
{
    llgo_cond_wait_call *call = (llgo_cond_wait_call *)raw;
    call->error = SleepConditionVariableSRW(
        call->condition, call->lock, call->milliseconds, 0)
        ? 0 : (int)GetLastError();
    return 0;
}

#if defined(LLGO_USE_BDWGC)
void *GC_do_blocking(void *(*fn)(void *), void *data);
#endif

static int llgo_cond_wait(llgo_condition_variable *condition,
                          llgo_srwlock *lock, llgo_dword milliseconds)
{
    llgo_cond_wait_call call = {condition, lock, milliseconds, 0};
    /* LLDB cannot unwind the optimized MSVC ARM64 GC_do_blocking frames:
     * https://github.com/llvm/llvm-project/issues/224600. Keep those threads
     * active in the collector so their Go stacks remain debuggable. */
#if defined(LLGO_USE_BDWGC) && !(defined(_MSC_VER) && defined(_M_ARM64))
    /* The native wait neither allocates GC memory nor runs Go code. Keep
     * its caller's roots visible without suspending this blocked thread
     * during every collection. GC_do_blocking restores the active state
     * before returning to Go. */
    GC_do_blocking(llgo_cond_wait_inactive, &call);
#else
    llgo_cond_wait_inactive(&call);
#endif
    return call.error;
}

int llgo_win_cond_wait(llgo_condition_variable *condition,
                       llgo_srwlock *lock)
{
    return llgo_cond_wait(condition, lock, LLGO_INFINITE);
}

/* Idle runtime worker threads carry no Go frames for LLDB to unwind, so
 * always park them outside BDWGC's active thread set (including on MSVC ARM64)
 * to avoid suspending idle threads inside ntdll wait frames during STW GC. */
int llgo_win_cond_wait_inactive(llgo_condition_variable *condition,
                                llgo_srwlock *lock)
{
    llgo_cond_wait_call call = {condition, lock, LLGO_INFINITE, 0};
#if defined(LLGO_USE_BDWGC)
    GC_do_blocking(llgo_cond_wait_inactive, &call);
#else
    llgo_cond_wait_inactive(&call);
#endif
    return call.error;
}

typedef struct {
    llgo_time_t sec;
    long nsec;
} llgo_timespec;

static unsigned long long llgo_unix_time_100ns(void)
{
    llgo_filetime now;
    unsigned long long ticks;
    GetSystemTimeAsFileTime(&now);
    ticks = ((unsigned long long)now.high << 32) | now.low;
    /* Number of 100ns intervals from 1601-01-01 to 1970-01-01. */
    return ticks - 116444736000000000ULL;
}

int llgo_win_cond_timedwait(llgo_condition_variable *condition,
                            llgo_srwlock *lock,
                            const llgo_timespec *abstime)
{
    unsigned long long deadline;
    unsigned long long now;
    unsigned long long remaining;
    llgo_dword milliseconds;
    llgo_dword error;

    if (abstime == 0 || abstime->sec < 0 || abstime->nsec < 0 ||
        abstime->nsec >= 1000000000L)
        return llgo_error_invalid_parameter;
    deadline = (unsigned long long)abstime->sec * 10000000ULL
             + (unsigned long long)abstime->nsec / 100ULL;
    now = llgo_unix_time_100ns();
    if (deadline <= now)
        return llgo_timedout;
    remaining = deadline - now;
    /* Round up so a sub-millisecond remainder cannot time out early. */
    remaining = (remaining + 9999ULL) / 10000ULL;
    milliseconds = remaining >= LLGO_INFINITE
                 ? LLGO_INFINITE - 1
                 : (llgo_dword)remaining;
    error = (llgo_dword)llgo_cond_wait(condition, lock, milliseconds);
    return error == llgo_error_timeout ? llgo_timedout : (int)error;
}

typedef struct {
    volatile unsigned int *address;
    unsigned int value;
    int error;
} llgo_wait_uint32_call;

static void *llgo_wait_uint32_inactive_cb(void *raw)
{
    llgo_wait_uint32_call *call = (llgo_wait_uint32_call *)raw;
    call->error = WaitOnAddress(
        call->address, &call->value, sizeof(call->value), LLGO_INFINITE)
        ? 0 : (int)GetLastError();
    return 0;
}

int llgo_win_wait_uint32(volatile unsigned int *address,
                         unsigned int value)
{
    llgo_wait_uint32_call call = {address, value, 0};
    /* Match llgo_cond_wait: keep MSVC ARM64 active for LLDB Go-frame unwinding. */
#if defined(LLGO_USE_BDWGC) && !(defined(_MSC_VER) && defined(_M_ARM64))
    GC_do_blocking(llgo_wait_uint32_inactive_cb, &call);
#else
    llgo_wait_uint32_inactive_cb(&call);
#endif
    return call.error;
}

/* Match llgo_win_cond_wait_inactive: detached idle worker threads carry no Go
 * frames, so park them outside BDWGC's active set on every Windows ABI. */
int llgo_win_wait_uint32_inactive(volatile unsigned int *address,
                                  unsigned int value)
{
    llgo_wait_uint32_call call = {address, value, 0};
#if defined(LLGO_USE_BDWGC)
    GC_do_blocking(llgo_wait_uint32_inactive_cb, &call);
#else
    llgo_wait_uint32_inactive_cb(&call);
#endif
    return call.error;
}
