#include <errno.h>
#include <stdint.h>
#include <stdio.h>
#include <time.h>

extern int32_t LLGoHostSyscallStatus(void);

int main(void)
{
    unsigned rejected = 0;
    for (unsigned i = 0; i < 50; i++) {
        int32_t status = LLGoHostSyscallStatus();
        if (status == 0) {
            fputs("C host thread was incorrectly treated as LLGo-owned: syscall succeeded\n", stderr);
            return 1;
        }
        if (status == ENOTSUP) rejected++;
        else if (status != EAGAIN) {
            fprintf(stderr, "unexpected all-thread syscall error: %d\n", status);
            return 2;
        }
        const struct timespec pause = {0, 1000000};
        nanosleep(&pause, 0);
    }
    if (!rejected) {
        fputs("C host never reached a stable ownership check\n", stderr);
        return 3;
    }
    puts("allthreads-adopted-host-ok");
    return 0;
}
