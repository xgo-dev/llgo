#include <pthread.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

uintptr_t llgo_test_callback_word(uintptr_t fn, uintptr_t value)
{
    return ((uintptr_t (*)(uintptr_t))fn)(value);
}

uintptr_t llgo_test_callback_integers(uintptr_t fn)
{
    typedef uintptr_t (*callback)(uintptr_t, uintptr_t, uintptr_t, uintptr_t,
        uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t,
        uintptr_t, uintptr_t, uintptr_t, uintptr_t, uintptr_t);
    return ((callback)fn)(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15);
}

uintptr_t llgo_test_callback_mixed(uintptr_t fn)
{
    typedef uintptr_t (*callback)(int8_t, float, uint16_t, double, int32_t, double,
        uint64_t, double, uintptr_t, double, int64_t, double, uintptr_t, double,
        uintptr_t, double, uintptr_t, double);
    return ((callback)fn)(-3, 0.25f, 60001, 1.5, -1234567, 2.5,
        UINT64_C(0x8000000000000007), 3.5, 17, 4.5, INT64_C(-1234567890123), 5.5,
        19, 6.5, 23, 7.5, 29, 8.5);
}

int32_t llgo_test_callback_narrow(uintptr_t signed_fn, uintptr_t unsigned_fn,
    uintptr_t bool_fn, uintptr_t pointer_fn, uint64_t *value)
{
    if (((int8_t (*)(int8_t))signed_fn)(-101) != -91)
        return 1;
    if (((uint16_t (*)(uint16_t))unsigned_fn)(60001) != 60002)
        return 2;
    if (!((bool (*)(bool))bool_fn)(false) || ((bool (*)(bool))bool_fn)(true))
        return 3;
    if (((uint64_t *(*)(uint64_t *))pointer_fn)(value) != value ||
        *value != UINT64_C(0x123456789abcdef0))
        return 4;
    if (((uint64_t *(*)(uint64_t *))pointer_fn)(NULL) != NULL)
        return 5;
    return 0;
}

int32_t llgo_test_callback_sort(uintptr_t ascending, uintptr_t descending)
{
    uint64_t data[] = {11, 1, 77, 3, 11, 5};
    qsort(data, 6, sizeof(data[0]), (int (*)(const void *, const void *))ascending);
    for (unsigned i = 1; i < 6; ++i) {
        if (data[i-1] > data[i])
            return 1;
    }
    qsort(data, 6, sizeof(data[0]), (int (*)(const void *, const void *))descending);
    for (unsigned i = 1; i < 6; ++i) {
        if (data[i-1] < data[i])
            return 2;
    }
    return 0;
}

typedef struct {
    uintptr_t fn;
    uintptr_t seed;
    int failed;
} thread_args;

static void *thread_main(void *raw)
{
    thread_args *args = raw;
    for (uintptr_t i = 0; i < 20; ++i) {
        if (llgo_test_callback_word(args->fn, i) != args->seed + i)
            args->failed = 1;
    }
    return NULL;
}

int32_t llgo_test_callback_thread(uintptr_t fn, uintptr_t seed)
{
    pthread_t thread;
    thread_args args = {fn, seed, 0};
    int error = pthread_create(&thread, NULL, thread_main, &args);
    if (error != 0)
        return error;
    error = pthread_join(thread, NULL);
    return error != 0 ? error : args.failed;
}

void llgo_test_callback_void(uintptr_t fn)
{
    ((void (*)(void))fn)();
}
