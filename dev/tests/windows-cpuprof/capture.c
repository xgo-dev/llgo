/* Deterministic regression for the production suspended-thread CONTEXT read.
 * Keep this as a separate native test: it must not depend on statistical CPU
 * samples, Go scheduling, or a Windows SDK CONTEXT definition. */
#include <stdio.h>

/* Including the implementation lets this test exercise its static helpers,
 * including the exact GetThreadContext flags used by llgo_prof_capture. */
#include "../../../runtime/internal/lib/runtime/_wrap/profile_windows.c"

__declspec(dllimport) void LLGO_WINAPI Sleep(llgo_dword milliseconds);
__declspec(dllimport) unsigned long long LLGO_WINAPI GetTickCount64(void);
__declspec(dllimport) llgo_dword LLGO_WINAPI GetLastError(void);

struct probe_state {
    int ready;
    int stop;
    llgo_uintptr frame_pointer;
    llgo_uintptr checksum;
};

/* All shared storage outlives the worker, including timeout cleanup. */
static struct probe_state state;

__attribute__((noinline)) static llgo_uintptr probe_leaf(void)
{
    volatile llgo_uintptr padding[256];
    llgo_uintptr checksum = 0;
    llgo_uintptr frame_pointer;
    unsigned int i;

    /* Keep a real, nontrivial frame. On Win64 RBP need not point at the
     * canonical saved-FP/return-PC pair, so this test checks the register
     * itself rather than assuming that an arbitrary FP chain can unwind it. */
    for (i = 0; i < 256; i++)
        padding[i] = (llgo_uintptr)i + 17;
    /* __builtin_frame_address can adjust the Windows frame register by its
     * unwind offset (for example, RBP-128). Compare CONTEXT with the actual
     * register, not that compiler-defined frame address. */
#if defined(_M_ARM64) || defined(__aarch64__)
    __asm__ volatile("mov %0, x29" : "=r"(frame_pointer));
#elif defined(_M_X64) || defined(__x86_64__)
    __asm__ volatile("movq %%rbp, %0" : "=r"(frame_pointer));
#else
    __asm__ volatile("movl %%ebp, %0" : "=r"(frame_pointer));
#endif
    state.frame_pointer = frame_pointer;
    __atomic_store_n(&state.ready, 1, __ATOMIC_RELEASE);

    /* No calls, waits, locks, or profiling helpers while ready. The controller
     * may safely suspend us anywhere in this loop with this same FP. */
    while (!__atomic_load_n(&state.stop, __ATOMIC_ACQUIRE)) {
    }

    for (i = 0; i < 256; i++)
        checksum += padding[i];
    return checksum;
}

__attribute__((noinline)) static llgo_uintptr probe_outer(void)
{
    volatile llgo_uintptr padding[64];
    llgo_uintptr result;
    unsigned int i;

    for (i = 0; i < 64; i++)
        padding[i] = (llgo_uintptr)i + 1;
    result = probe_leaf();
    /* Work after the call also prevents a tail-call-only outer frame. */
    return result + padding[63];
}

static llgo_dword LLGO_WINAPI probe_worker(void *arg)
{
    (void)arg;
    state.checksum = probe_outer();
    return 0;
}

int main(void)
{
#if LLGO_PROF_CONTEXT_SUPPORTED
    unsigned char storage[LLGO_PROF_CONTEXT_SIZE + 15];
    unsigned char *context = (unsigned char *)
        (((llgo_uintptr)(storage + 15)) & ~(llgo_uintptr)15);
    struct llgo_prof_sample sample;
    llgo_handle thread;
    unsigned long long deadline;
    llgo_uintptr actual_fp = 0;
    llgo_uintptr actual_pc = 0;
    llgo_dword context_error = 0;
    llgo_dword capture_error = 0;
    llgo_dword context_flags = 0;
    int got_context = 0;
    int got_sample = 0;
    int suspended = 0;
    int failed = 0;

    if (llgo_cpu_profile_test_fault_recovery() != 1) {
        fprintf(stderr, "unaligned invalid FP did not preserve the interrupted PC\n");
        return 1;
    }
    memset(&sample, 0, sizeof(sample));
    sample.n = 1;
    sample.pc[0] = 12345;
    /* Unlike FP=1, this passes the alignment check and exercises the guarded
     * ReadProcessMemory failure on an inaccessible low address. */
    llgo_prof_walk_frames(&sample, 16);
    if (sample.n != 1 || sample.pc[0] != 12345) {
        fprintf(stderr, "unreadable aligned FP corrupted the interrupted PC\n");
        return 1;
    }

    thread = CreateThread(0, 0, probe_worker, 0, 0, 0);
    if (thread == 0) {
        fprintf(stderr, "CreateThread failed: %lu\n", GetLastError());
        return 1;
    }
    deadline = GetTickCount64() + 5000;
    while (!__atomic_load_n(&state.ready, __ATOMIC_ACQUIRE)) {
        if (GetTickCount64() >= deadline) {
            fprintf(stderr, "worker did not become ready within 5 seconds\n");
            failed = 1;
            goto cleanup;
        }
        Sleep(1);
    }
    if (SuspendThread(thread) == LLGO_SUSPEND_FAILED) {
        fprintf(stderr, "SuspendThread failed: %lu\n", GetLastError());
        failed = 1;
        goto cleanup;
    }
    suspended = 1;

    /* Do not print or run assertions while the worker is suspended. Always
     * resume it even if either production context operation fails. */
    got_context = llgo_prof_get_context(thread, context);
    if (got_context) {
        memcpy(&context_flags, context + LLGO_PROF_CONTEXT_FLAGS_OFFSET,
               sizeof(context_flags));
        actual_fp = llgo_prof_context_word(context, LLGO_PROF_CONTEXT_FP_OFFSET);
        actual_pc = llgo_prof_context_word(context, LLGO_PROF_CONTEXT_PC_OFFSET);
    } else {
        context_error = GetLastError();
    }
    memset(&sample, 0, sizeof(sample));
    got_sample = llgo_prof_capture(thread, &sample);
    if (!got_sample)
        capture_error = GetLastError();
    if (ResumeThread(thread) == LLGO_SUSPEND_FAILED) {
        fprintf(stderr, "ResumeThread failed: %lu\n", GetLastError());
        failed = 1;
        goto cleanup;
    }
    suspended = 0;

    printf("context FP=%p, worker FP=%p, PC=%p, sample frames=%u, flags=%#lx\n",
           (void *)actual_fp, (void *)state.frame_pointer, (void *)actual_pc,
           (unsigned int)sample.n, context_flags);
    if (!got_context) {
        fprintf(stderr, "production GetThreadContext failed: %lu\n", context_error);
        failed = 1;
    } else if (actual_fp == 0 || actual_fp != state.frame_pointer || actual_pc < 4096) {
        fprintf(stderr, "production CONTEXT did not preserve the worker's real FP and PC\n");
        failed = 1;
    }
#if defined(_WIN64) && (defined(_M_X64) || defined(__x86_64__))
    /* Some Windows versions/emulators fill unrequested registers too. Do not
     * let that incidental behavior hide a missing CONTEXT_INTEGER request. */
    if (got_context && (context_flags & 0x00100003UL) != 0x00100003UL) {
        fprintf(stderr, "AMD64 frame-pointer capture did not request CONTEXT_INTEGER\n");
        failed = 1;
    }
#endif
    if (!got_sample || sample.n == 0 || sample.pc[0] < 4096 ||
        (got_context && sample.pc[0] != actual_pc + 1)) {
        fprintf(stderr, "production capture did not preserve an interrupted PC (error=%lu)\n",
                capture_error);
        failed = 1;
    }

cleanup:
    if (suspended && ResumeThread(thread) == LLGO_SUSPEND_FAILED) {
        fprintf(stderr, "cleanup ResumeThread failed: %lu\n", GetLastError());
        failed = 1;
    }
    __atomic_store_n(&state.stop, 1, __ATOMIC_RELEASE);
    if (WaitForSingleObject(thread, 5000) != llgo_wait_object_0) {
        fprintf(stderr, "worker did not exit within 5 seconds\n");
        failed = 1;
    } else if (state.checksum != 37056) {
        fprintf(stderr, "worker stack-local checksum was not preserved\n");
        failed = 1;
    }
    if (!CloseHandle(thread)) {
        fprintf(stderr, "CloseHandle failed: %lu\n", GetLastError());
        failed = 1;
    }
    if (!failed)
        puts("Windows CPU profiler CONTEXT regression: PASS");
    return failed;
#else
    fprintf(stderr, "unsupported Windows CPU profiler architecture\n");
    return 1;
#endif
}
