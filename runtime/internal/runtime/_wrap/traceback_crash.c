#include <stdlib.h>
#include <stdint.h>

#if defined(_WIN32)
#if defined(_WIN64)
#define LLGO_WINAPI
#else
#define LLGO_WINAPI __attribute__((stdcall))
#endif
__declspec(dllimport) unsigned long LLGO_WINAPI GetErrorMode(void);
__declspec(dllimport) unsigned long LLGO_WINAPI SetErrorMode(unsigned long);
__declspec(dllimport) void LLGO_WINAPI RaiseFailFastException(void *, void *, unsigned long);
__declspec(dllimport) long LLGO_WINAPI WerGetFlags(void *, unsigned long *);
__declspec(dllimport) long LLGO_WINAPI WerSetFlags(unsigned long);
#elif !defined(__wasm__)
#include <signal.h>
#endif

void llgo_traceback_config(_Bool initialize, _Bool wer)
{
#if defined(_WIN32)
    if (initialize) {
        /* Match Go's preventErrorDialogs; preserve the embedding host's other
         * flags. Enabling WER later must not enable interactive fault UI. */
        SetErrorMode(GetErrorMode() | 1 | 2 | 0x8000);
        unsigned long flags = 0;
        WerGetFlags((void *)(intptr_t)-1, &flags);
        WerSetFlags(flags | 0x20); /* WER_FAULT_REPORTING_NO_UI */
    }
    /* This process-wide change is sticky, as in Go's enableWER. A subsequent
     * SetTraceback("all") changes detail, without disabling WER again. */
    if (wer)
        SetErrorMode(GetErrorMode() & ~2UL);
#else
    (void)initialize; (void)wer;
#endif
}

void llgo_traceback_crash(void)
{
#if defined(_WIN32)
    struct {
        unsigned long code, flags;
        void *nested, *address;
        unsigned long count;
        uintptr_t information[15];
    } record = {0};
    record.code = 2; /* Go's ordinary panic exception status. */
    RaiseFailFastException(&record, 0, 1);
#elif !defined(__wasm__)
    signal(SIGABRT, SIG_DFL);
    raise(SIGABRT);
#endif
    abort();
}
