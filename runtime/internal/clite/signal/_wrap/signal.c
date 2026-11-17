#if !defined(_WIN32) && !defined(__wasm__) && !defined(__ESP__)
#define _POSIX_C_SOURCE 200809L
#include <errno.h>
#include <signal.h>

int llgo_core_signal(int sig, void (*handler)(int))
{
    int saved_errno = errno;
    struct sigaction act = {0};
    act.sa_handler = handler;
    sigemptyset(&act.sa_mask);
    /* Recovery uses savemask=0 and leaves via siglongjmp, bypassing
     * sigreturn. Otherwise the first fault leaves its signal blocked. */
    if (sig == SIGSEGV || sig == SIGBUS)
        act.sa_flags = SA_NODEFER;
    int result = sigaction(sig, &act, 0);
    if (result == 0)
        errno = saved_errno;
    return result;
}
#endif
