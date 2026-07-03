#include <unistd.h>

int llgo_maxprocs()
{
#ifdef _SC_NPROCESSORS_ONLN
    return (int)sysconf(_SC_NPROCESSORS_ONLN);
#else
    return 1;
#endif
}

__attribute__((noinline)) void *llgo_framepointer(void)
{
#if defined(__GNUC__) || defined(__clang__)
    return __builtin_frame_address(0);
#else
    return 0;
#endif
}
