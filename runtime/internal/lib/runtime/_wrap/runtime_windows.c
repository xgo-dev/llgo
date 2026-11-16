/* Keep the runtime shim independent of Windows SDK headers. Clang still
 * applies the target's MSVC ABI and emits ordinary Kernel32 imports. */
#include <errno.h>
#include <stdint.h>

typedef __SIZE_TYPE__ llgo_size_t;
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
} llgo_system_info;

typedef struct {
    void *base_address;
    void *allocation_base;
    llgo_dword allocation_protect;
#if defined(_WIN64)
    llgo_word partition_id;
#endif
    llgo_size_t region_size;
    llgo_dword state;
    llgo_dword protect;
    llgo_dword type;
} llgo_memory_basic_information;

__declspec(dllimport) int LLGO_WINAPI
GetProcessAffinityMask(void *process, llgo_uintptr *process_mask,
                       llgo_uintptr *system_mask);
__declspec(dllimport) void LLGO_WINAPI GetSystemInfo(llgo_system_info *info);
__declspec(dllimport) llgo_size_t LLGO_WINAPI
VirtualQuery(const void *address, llgo_memory_basic_information *info,
             llgo_size_t length);
__declspec(dllimport) int LLGO_WINAPI
QueryPerformanceCounter(long long *counter);
__declspec(dllimport) int LLGO_WINAPI
QueryPerformanceFrequency(long long *frequency);

static long long llgo_nanotime_frequency;

/* C2func wrappers read the calling thread's CRT errno through this common
 * runtime entry point. Unix provides the same symbol from clite/os. */
int cliteErrno(void)
{
    return errno;
}

typedef struct {
    llgo_dword low;
    llgo_dword high;
} llgo_filetime;

__declspec(dllimport) void LLGO_WINAPI
GetSystemTimePreciseAsFileTime(llgo_filetime *time);
__declspec(dllimport) void *LLGO_WINAPI
LoadLibraryExW(const llgo_word *filename, void *file, llgo_dword flags);
__declspec(dllimport) void *LLGO_WINAPI
GetProcAddress(void *module, const char *name);
__declspec(dllimport) llgo_dword LLGO_WINAPI GetLastError(void);

#if defined(_WIN64)
/* RUNTIME_FUNCTION differs between AMD64 and ARM64; only its pointer is used. */
typedef struct llgo_runtime_function llgo_runtime_function;

__declspec(dllimport) void LLGO_WINAPI RtlCaptureContext(void *context);
__declspec(dllimport) llgo_runtime_function *LLGO_WINAPI
RtlLookupFunctionEntry(llgo_uintptr pc, llgo_uintptr *image_base,
                       void *history_table);
__declspec(dllimport) void *LLGO_WINAPI
RtlVirtualUnwind(llgo_dword handler_type, llgo_uintptr image_base,
                 llgo_uintptr pc, llgo_runtime_function *function_entry,
                 void *context, void **handler_data,
                 llgo_uintptr *establisher_frame, void *context_pointers);

__attribute__((noinline)) int
llgo_windows_capture_context(void *context, llgo_size_t pc_offset)
{
    llgo_runtime_function *function_entry;
    llgo_uintptr image_base;
    llgo_uintptr pc;
    llgo_uintptr frame;
    void *handler_data;

    RtlCaptureContext(context);
    /* Unwind this wrapper while its captured stack frame is still live. */
    pc = *(llgo_uintptr *)((unsigned char *)context + pc_offset);
    function_entry = RtlLookupFunctionEntry(pc, &image_base, 0);
    if (function_entry == 0)
        return 0;
    /* HandlerData is a required output on ARM64, even for a no-handler walk. */
    RtlVirtualUnwind(0, image_base, pc, function_entry, context,
                     &handler_data, &frame, 0);
    return 1;
}

llgo_runtime_function *
llgo_windows_lookup_function_entry(llgo_uintptr pc, llgo_uintptr *image_base)
{
    return RtlLookupFunctionEntry(pc, image_base, 0);
}

void llgo_windows_virtual_unwind(llgo_uintptr image_base, llgo_uintptr pc,
                               llgo_runtime_function *function_entry,
                               void *context,
                               llgo_uintptr *establisher_frame)
{
    void *handler_data;
    /* HandlerData is a required output on ARM64, even for a no-handler walk. */
    RtlVirtualUnwind(0, image_base, pc, function_entry, context,
                     &handler_data, establisher_frame, 0);
}
#endif

enum {
    llgo_mem_commit = 0x1000,
    llgo_page_noaccess = 0x01,
    llgo_page_execute = 0x10,
    llgo_page_guard = 0x100,
};

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
        llgo_system_info info;
        GetSystemInfo(&info);
        return info.num_processors == 0 ? 1 : (int)info.num_processors;
    }
}

int llgo_mem_readable(void *p)
{
    llgo_memory_basic_information info;
    llgo_dword protect;
    if (p == 0 || VirtualQuery(p, &info, sizeof(info)) == 0 ||
        info.state != llgo_mem_commit)
        return 0;
    protect = info.protect & 0xff;
    return protect != llgo_page_noaccess && protect != llgo_page_execute &&
           (info.protect & llgo_page_guard) == 0;
}

int llgo_nanotime_init(void)
{
    return QueryPerformanceFrequency(&llgo_nanotime_frequency) &&
           llgo_nanotime_frequency > 0;
}

long long llgo_nanotime(void)
{
    long long counter;
    long long seconds;
    long long remainder;
    if (!QueryPerformanceCounter(&counter) || llgo_nanotime_frequency <= 0)
        return 0;
    seconds = counter / llgo_nanotime_frequency;
    remainder = counter % llgo_nanotime_frequency;
    return seconds * 1000000000LL +
           remainder * 1000000000LL / llgo_nanotime_frequency;
}

long long llgo_query_performance_counter(void)
{
    long long counter;
    if (!QueryPerformanceCounter(&counter))
        return 0;
    return counter;
}

long long llgo_query_performance_frequency(void)
{
    long long frequency;
    if (!QueryPerformanceFrequency(&frequency))
        return 0;
    return frequency;
}

void llgo_walltime(long long *seconds, long *nanoseconds)
{
    llgo_filetime now;
    unsigned long long ticks;
    GetSystemTimePreciseAsFileTime(&now);
    ticks = ((unsigned long long)now.high << 32) | now.low;
    ticks -= 116444736000000000ULL;
    *seconds = (long long)(ticks / 10000000ULL);
    *nanoseconds = (long)((ticks % 10000000ULL) * 100ULL);
}

llgo_uintptr llgo_load_library(const llgo_word *filename, llgo_dword flags,
                               llgo_dword *error)
{
    void *module = LoadLibraryExW(filename, 0, flags);
    *error = module == 0 ? GetLastError() : 0;
    return (llgo_uintptr)module;
}

llgo_uintptr llgo_get_proc_address(llgo_uintptr module,
                                   const unsigned char *name,
                                   llgo_dword *error)
{
    void *proc = GetProcAddress((void *)module, (const char *)name);
    *error = proc == 0 ? GetLastError() : 0;
    return (llgo_uintptr)proc;
}

/* --- Standard library OS bridges (link_windows_llgo.go) ------------------- */

typedef int (LLGO_WINAPI *llgo_console_handler)(llgo_dword event);
__declspec(dllimport) int LLGO_WINAPI
SetConsoleCtrlHandler(llgo_console_handler handler, int add);
__declspec(dllimport) void LLGO_WINAPI Sleep(llgo_dword milliseconds);
typedef struct llgo_overlapped llgo_overlapped;
__declspec(dllimport) void *LLGO_WINAPI
CreateIoCompletionPort(void *file, void *existing_port,
                       llgo_uintptr completion_key,
                       llgo_dword concurrent_threads);
__declspec(dllimport) int LLGO_WINAPI
GetQueuedCompletionStatus(void *port, llgo_dword *bytes,
                          llgo_uintptr *completion_key,
                          llgo_overlapped **overlapped,
                          llgo_dword milliseconds);

extern int llgo_runtime_windowsSignalCallback(llgo_dword signum);

enum {
    llgo_ctrl_c_event = 0,
    llgo_ctrl_break_event = 1,
    llgo_ctrl_close_event = 2,
    llgo_ctrl_logoff_event = 5,
    llgo_ctrl_shutdown_event = 6,
    llgo_sigint = 2,
    llgo_sigterm = 15,
};

static int LLGO_WINAPI llgo_windows_console_handler(llgo_dword event)
{
    llgo_dword signum;
    int handled;
    switch (event) {
    case llgo_ctrl_c_event:
    case llgo_ctrl_break_event:
        signum = llgo_sigint;
        break;
    case llgo_ctrl_close_event:
    case llgo_ctrl_logoff_event:
    case llgo_ctrl_shutdown_event:
        signum = llgo_sigterm;
        break;
    default:
        return 0;
    }
    handled = llgo_runtime_windowsSignalCallback(signum);
    if (!handled)
        return 0;
    if (signum == llgo_sigterm) {
        /* Windows terminates the process after a close, logoff, or shutdown
         * handler returns. Match Go's ctrlHandler by parking this dedicated
         * callback thread after SIGTERM has been accepted, leaving the other
         * LLGo native threads free to run os/signal cleanup until the process
         * exits. */
        for (;;)
            Sleep(0xffffffffUL);
    }
    return 1;
}

int llgo_getpagesize(void)
{
    llgo_system_info info;
    GetSystemInfo(&info);
    return (int)info.page_size;
}

int llgo_windows_signal_init(void)
{
    return SetConsoleCtrlHandler(llgo_windows_console_handler, 1);
}

llgo_uintptr llgo_iocp_create(llgo_dword *error)
{
    void *port = CreateIoCompletionPort((void *)(llgo_uintptr)-1, 0, 0, 0);
    *error = port == 0 ? GetLastError() : 0;
    return (llgo_uintptr)port;
}

int llgo_iocp_associate(llgo_uintptr port, llgo_uintptr handle,
                        llgo_uintptr key, llgo_dword *error)
{
    void *result = CreateIoCompletionPort((void *)handle, (void *)port, key, 0);
    *error = result == 0 ? GetLastError() : 0;
    return result != 0;
}

int llgo_iocp_get(llgo_uintptr port, llgo_uintptr *key,
                  llgo_overlapped **overlapped, llgo_dword *error)
{
    llgo_dword bytes;
    *overlapped = 0;
    int ok = GetQueuedCompletionStatus((void *)port, &bytes, key, overlapped,
                                       0xffffffffUL);
    if (!ok && *overlapped == 0) {
        *error = GetLastError();
        return 0;
    }
    /* A failed overlapped operation still produces a completion packet. The
     * caller obtains its operation-specific error with WSA/GetOverlappedResult. */
    *error = ok ? 0 : GetLastError();
    return 1;
}
