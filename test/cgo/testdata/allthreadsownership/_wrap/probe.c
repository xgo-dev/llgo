#include <time.h>

/* Test-only activation of the private protocol. The C host and executable
 * supply a controlled cohort; no arbitrary C library creates threads here. */
int __llgo_allthreads_executable = 1;

void llgo_test_allthreads_pause(void)
{
    const struct timespec pause = {0, 1000000};
    nanosleep(&pause, 0);
}
