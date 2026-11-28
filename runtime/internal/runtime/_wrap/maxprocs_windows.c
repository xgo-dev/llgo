/* Keep the runtime shim independent of Windows SDK headers. Clang still
 * applies the target's MSVC ABI and emits ordinary Kernel32 imports. */
#include <stdint.h>

typedef unsigned long llgo_dword;
typedef unsigned short llgo_word;
typedef __UINTPTR_TYPE__ llgo_uintptr;

#if defined(_WIN64)
#define LLGO_WINAPI
#else
#define LLGO_WINAPI __attribute__((stdcall))
#endif

typedef struct {
    union { llgo_dword oemid; struct { llgo_word arch; llgo_word reserved; }; };
    llgo_dword page_size;
    void *min_app_address;
    void *max_app_address;
    llgo_uintptr active_processor_mask;
    llgo_dword num_processors;
    llgo_dword processor_type;
    llgo_dword allocation_granularity;
    llgo_word processor_level;
    llgo_word processor_revision;
} llgo_maxprocs_system_info;

__declspec(dllimport) int LLGO_WINAPI
GetProcessAffinityMask(void *process, llgo_uintptr *process_mask,
                       llgo_uintptr *system_mask);
__declspec(dllimport) void LLGO_WINAPI
GetSystemInfo(llgo_maxprocs_system_info *info);

int llgo_maxprocs(void)
{
    llgo_uintptr mask;
    llgo_uintptr system_mask;
    int count = 0;

    /* Match Go's getCPUCount: report CPUs available to this process, rather
     * than all active processors on the machine. */
    if (GetProcessAffinityMask((void *)(intptr_t)-1, &mask, &system_mask)) {
        while (mask != 0) {
            count += (int)(mask & 1);
            mask >>= 1;
        }
        if (count != 0)
            return count;
    }
    {
        llgo_maxprocs_system_info info;
        GetSystemInfo(&info);
        return info.num_processors == 0 ? 1 : (int)info.num_processors;
    }
}
