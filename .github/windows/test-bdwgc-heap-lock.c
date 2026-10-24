/* A collector must not allocate CRT memory while another thread is stopped
 * with the Windows process heap locked. BDWGC's MinGW signal-based root-scan
 * guard can do exactly that: https://github.com/bdwgc/bdwgc/issues/924.
 *
 * Create both registered threads before locking the heap, and collect on a
 * fresh thread whose CRT signal table has not been allocated yet. */
#define GC_THREADS
#define GC_NO_THREAD_REDIRECTS
#include <gc/gc.h>
#include <windows.h>
#include <process.h>
#include <signal.h>
#include <stdio.h>

static HANDLE lock_start, locked, collector_ready, collect_start, release_heap;
static volatile LONG lock_requested;
static void *client_page;
static volatile LONG client_faults;

static LONG CALLBACK client_read_fault(PEXCEPTION_POINTERS info)
{
    DWORD previous;
    if (info->ExceptionRecord->ExceptionCode == EXCEPTION_ACCESS_VIOLATION
        && info->ExceptionRecord->ExceptionInformation[0] == 0
        && info->ExceptionRecord->ExceptionInformation[1] == (ULONG_PTR)client_page
        && VirtualProtect(client_page, 4096, PAGE_READWRITE, &previous)) {
        InterlockedIncrement(&client_faults);
        return EXCEPTION_CONTINUE_EXECUTION;
    }
    return EXCEPTION_CONTINUE_SEARCH;
}

static int check_client_handler(void)
{
    DWORD previous;
    PVOID handler;
    int ok;
    client_page = VirtualAlloc(NULL, 4096, MEM_RESERVE | MEM_COMMIT, PAGE_READWRITE);
    if (!client_page)
        return 0;
    *(volatile int *)client_page = 42;
    /* Append after the collector's handler: it must leave this fault alone. */
    handler = AddVectoredExceptionHandler(0, client_read_fault);
    if (!handler || !VirtualProtect(client_page, 4096, PAGE_NOACCESS, &previous))
        return 0;
    ok = *(volatile int *)client_page == 42 && client_faults == 1;
    RemoveVectoredExceptionHandler(handler);
    VirtualFree(client_page, 0, MEM_RELEASE);
    return ok;
}

static unsigned __stdcall hold_heap(void *unused)
{
    DWORD deadline;
    (void)unused;
    WaitForSingleObject(lock_start, INFINITE);
    if (!HeapLock(GetProcessHeap()))
        TerminateProcess(GetCurrentProcess(), 3);
    SetEvent(locked);
    /* A real Windows heap operation eventually releases the lock while
     * runnable. Suspension during this bounded interval must not deadlock GC. */
    deadline = GetTickCount() + 1000;
    while ((LONG)(GetTickCount() - deadline) < 0
           && WaitForSingleObject(release_heap, 0) != WAIT_OBJECT_0)
        SwitchToThread();
    HeapUnlock(GetProcessHeap());
    return 0;
}

static void client_signal_handler(int signal_number)
{
    (void)signal_number;
}

static void GC_CALLBACK collection_event(GC_EventType event)
{
    /* Heap discovery before this event may itself take the process heap lock.
     * Acquire it immediately before suspension so that preliminary GC work
     * cannot hide the lazy signal-table allocation during root scanning. */
    if (event == GC_EVENT_PRE_STOP_WORLD
        && InterlockedCompareExchange(&lock_requested, 1, 0) == 0) {
        SetEvent(lock_start);
        if (WaitForSingleObject(locked, 10000) != WAIT_OBJECT_0)
            TerminateProcess(GetCurrentProcess(), 5);
    }
}

static unsigned __stdcall collect(void *unused)
{
    (void)unused;
    SetEvent(collector_ready);
    WaitForSingleObject(collect_start, INFINITE);
    GC_gcollect();
    SetEvent(release_heap);
    /* Preparation and scanning must both restore the caller's CRT handler. */
    if (signal(SIGSEGV, client_signal_handler) == SIG_ERR)
        TerminateProcess(GetCurrentProcess(), 8);
    GC_gcollect();
    if (signal(SIGSEGV, SIG_DFL) != client_signal_handler)
        TerminateProcess(GetCurrentProcess(), 9);
    return 0;
}

int main(void)
{
    HANDLE threads[2];
    GC_INIT();
    if (!check_client_handler())
        return 7;
    lock_start = CreateEventW(NULL, TRUE, FALSE, NULL);
    locked = CreateEventW(NULL, TRUE, FALSE, NULL);
    collector_ready = CreateEventW(NULL, TRUE, FALSE, NULL);
    collect_start = CreateEventW(NULL, TRUE, FALSE, NULL);
    release_heap = CreateEventW(NULL, TRUE, FALSE, NULL);
    threads[0] = (HANDLE)GC_beginthreadex(NULL, 0, hold_heap, NULL, 0, NULL);
    threads[1] = (HANDLE)GC_beginthreadex(NULL, 0, collect, NULL, 0, NULL);
    if (!threads[0] || !threads[1])
        return 4;
    if (WaitForSingleObject(collector_ready, 10000) != WAIT_OBJECT_0)
        return 5;
    GC_set_on_collection_event(collection_event);
    SetEvent(collect_start);
    if (WaitForMultipleObjects(2, threads, TRUE, 10000) != WAIT_OBJECT_0)
        TerminateProcess(GetCurrentProcess(), 6);
    CloseHandle(threads[0]);
    CloseHandle(threads[1]);
    CloseHandle(lock_start);
    CloseHandle(locked);
    CloseHandle(collector_ready);
    CloseHandle(collect_start);
    CloseHandle(release_heap);
    puts("GC completed while another registered thread held the process heap lock");
    return 0;
}
