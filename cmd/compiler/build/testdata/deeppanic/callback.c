extern void panic_callback(void);
static volatile int callback_returned;

void call_go(void) {
    panic_callback();
    callback_returned++;
}

#include <stdlib.h>
#if defined(_WIN32)
#include <windows.h>
static DWORD WINAPI callback_thread(LPVOID unused) {
    (void)unused;
    call_go();
    return 0;
}
void call_go_on_thread(void) {
    HANDLE t = CreateThread(0, 0, callback_thread, 0, 0, 0);
    if (!t) abort();
    WaitForSingleObject(t, INFINITE);
    CloseHandle(t);
}
#else
#include <pthread.h>
#include <dlfcn.h>
static void *callback_thread(void *unused) {
    (void)unused;
    call_go();
    return 0;
}
void call_go_on_thread(void) {
    pthread_t t;
    if (pthread_create(&t, 0, callback_thread, 0)) abort();
    pthread_join(t, 0);
}
#endif

void call_go_through_library(void) {
    const char *path = getenv("LLGO_TRACEBACK_CALLBACK_LIBRARY");
    void (*bridge)(void (*)(void));
#if defined(_WIN32)
    HMODULE library = LoadLibraryA(path);
    if (!library) abort();
    bridge = (void (*)(void (*)(void)))GetProcAddress(library, "callback_bridge");
#else
    void *library = dlopen(path, RTLD_NOW | RTLD_LOCAL);
    if (!library) abort();
    bridge = (void (*)(void (*)(void)))dlsym(library, "callback_bridge");
#endif
    if (!bridge) abort();
    bridge(panic_callback);
    callback_returned++;
}
