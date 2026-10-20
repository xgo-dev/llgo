#include <stdint.h>
#if !defined(__wasm__)
#if defined(_WIN32)
#include <windows.h>
#else
#include <pthread.h>
#endif

int32_t llgo_ext_counter = 41;
int32_t llgo_ext_array[3] = {2, 3, 5};
int32_t *llgo_ext_pointer = &llgo_ext_counter;
static int32_t twice(int32_t v) { return v * 2; }
int32_t (*llgo_ext_callback)(int32_t) = twice;
int32_t llgo_ext_addresses(void *counter, void *array, void *pointer, void *callback) {
    return counter == &llgo_ext_counter && array == &llgo_ext_array &&
        pointer == &llgo_ext_pointer && callback == &llgo_ext_callback;
}
extern int32_t llgo_ext_go_entry(int32_t);
extern int32_t llgo_ext_go_callback(int32_t (*)(int32_t), int32_t);
int32_t llgo_ext_invoke(int32_t v) { return llgo_ext_go_entry(v); }
int32_t llgo_ext_invoke_callback(int32_t v) { return llgo_ext_go_callback(twice, v); }

struct thread_args { int32_t value; int32_t result; };
#if defined(_WIN32)
static DWORD WINAPI entry(void *opaque) {
#else
static void *entry(void *opaque) {
#endif
    struct thread_args *args = opaque;
    args->result = llgo_ext_go_entry(args->value);
    return 0;
}
int32_t llgo_ext_thread(int32_t v) {
    struct thread_args args = {v, -1};
#if defined(_WIN32)
    HANDLE thread = CreateThread(0, 0, entry, &args, 0, 0);
    if (!thread) return -2;
    WaitForSingleObject(thread, INFINITE);
    CloseHandle(thread);
#else
    pthread_t thread;
    if (pthread_create(&thread, 0, entry, &args) != 0) return -2;
    pthread_join(thread, 0);
#endif
    return args.result;
}
#endif

#if defined(_WIN32)
extern int32_t __stdcall llgo_ext_std_entry(int32_t);
int32_t __stdcall llgo_ext_std_twice(int32_t v) { return v * 2; }
int32_t (__stdcall *llgo_ext_std_callback)(int32_t) = llgo_ext_std_twice;
int32_t llgo_ext_invoke_std(int32_t v) { return llgo_ext_std_entry(v); }
#endif
