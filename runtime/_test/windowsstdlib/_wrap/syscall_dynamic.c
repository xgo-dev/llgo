#include <stdarg.h>
#include <stdint.h>
#include <windows.h>

static int unwind_seen;
extern void llgo_windows_syscall(void *);

static void check_unwind(void)
{
    void *frames[32];
    USHORT count = CaptureStackBackTrace(0, 32, frames, NULL);
    unwind_seen = 0;
#if defined(__x86_64__) || defined(__aarch64__)
    for (USHORT i = 0; i < count; i++) {
        DWORD64 image = 0;
        PRUNTIME_FUNCTION function = RtlLookupFunctionEntry((DWORD64)frames[i], &image, NULL);
        if (function && image + function->BeginAddress == (DWORD64)(uintptr_t)llgo_windows_syscall)
            unwind_seen = 1;
    }
#else
    /* x86 has an EBP chain rather than x64/ARM64 function-table entries. */
    unwind_seen = count >= 3;
#endif
}

int llgo_windows_unwind_seen(void)
{
    return unwind_seen;
}

uintptr_t llgo_windows_no_arguments(void)
{
    if (GetLastError() != 0)
        __builtin_trap();
    check_unwind();
    SetLastError(1234);
    return 7;
}

uintptr_t llgo_windows_sum_variable(uintptr_t count, ...)
{
    if (GetLastError() != 0)
        __builtin_trap();
    va_list arguments;
    va_start(arguments, count);
    uintptr_t result = 0;
    for (uintptr_t i = 0; i < count; i++)
        result += va_arg(arguments, uintptr_t);
    va_end(arguments);
    check_unwind();
    SetLastError(1234);
    return result;
}

uint64_t llgo_windows_high_word(void)
{
    return UINT64_C(0x8123456789abcdef);
}

#if defined(__i386__)
#define LLGO_STDCALL __attribute__((stdcall))
#else
#define LLGO_STDCALL
#endif

uintptr_t LLGO_STDCALL llgo_windows_sum12_stdcall(
    uintptr_t a1, uintptr_t a2, uintptr_t a3, uintptr_t a4,
    uintptr_t a5, uintptr_t a6, uintptr_t a7, uintptr_t a8,
    uintptr_t a9, uintptr_t a10, uintptr_t a11, uintptr_t a12)
{
    return a1 + a2 + a3 + a4 + a5 + a6 + a7 + a8 + a9 + a10 + a11 + a12;
}

uint32_t llgo_windows_call_stdcall_address(uintptr_t address, void *info)
{
    return ((uint32_t (*)(void *))address)(info);
}
