#if defined(__linux__) && !defined(_GNU_SOURCE)
#define _GNU_SOURCE 1
#endif
#if defined(_WIN32)
#define _CRT_SECURE_NO_WARNINGS 1
#endif
#include <assert.h>
#include <stdio.h>
#include <string.h>
#include "../_wrap/gc_policy.c"

extern void GC_init(void);
extern void GC_gcollect(void);
extern uintptr_t GC_get_gc_no(void);
extern void *GC_malloc_atomic(size_t);
extern int GC_get_disable_automatic_collection(void);
extern uintptr_t GC_get_free_space_divisor(void);
extern int GC_get_parallel(void);

static unsigned previous_callbacks;
static void previous_event(int event) { if (event == 5) ++previous_callbacks; }
static void *volatile sink;
static void *volatile retained[2048];

int main(int argc, char **argv)
{
    assert(llgo_gc_budget(0, 100) == 4u << 20);
    assert(llgo_gc_budget(8u << 20, 100) == 8u << 20);
    assert(llgo_gc_budget(8u << 20, 50) == 4u << 20);
    assert(llgo_gc_budget(SIZE_MAX, 200) == SIZE_MAX);
    assert(llgo_gc_budget(SIZE_MAX, INT32_MAX) == SIZE_MAX);
    assert(llgo_gc_budget(SIZE_MAX, 100) == SIZE_MAX);
    assert(llgo_gc_budget(0, 0) == 1);

    assert(llgo_gc_parse_gomaxprocs(NULL) == 0);
    assert(llgo_gc_parse_gomaxprocs("") == 0);
    assert(llgo_gc_parse_gomaxprocs("0") == 0);
    assert(llgo_gc_parse_gomaxprocs("-1") == 0);
    assert(llgo_gc_parse_gomaxprocs("+4") == 0);
    assert(llgo_gc_parse_gomaxprocs(" 4") == 0);
    assert(llgo_gc_parse_gomaxprocs("4 ") == 0);
    assert(llgo_gc_parse_gomaxprocs("bad") == 0);
    assert(llgo_gc_parse_gomaxprocs("2147483648") == 0);
    assert(llgo_gc_parse_gomaxprocs("1") == 1);
    assert(llgo_gc_parse_gomaxprocs("4") == 4);
    assert(llgo_gc_parse_gomaxprocs("2147483647") == (unsigned)INT32_MAX);

    assert(llgo_gc_target_markers(NULL, NULL, NULL, 16) == 0);
    assert(llgo_gc_target_markers("1", NULL, NULL, 16) == 1);
    assert(llgo_gc_target_markers("4", "", "", 16) == 4);
    assert(llgo_gc_target_markers("64", NULL, NULL, 16) == 16);
    assert(llgo_gc_target_markers("4", "8", NULL, 16) == 0);
    assert(llgo_gc_target_markers("4", NULL, "8", 16) == 0);
    assert(llgo_gc_target_markers("0", NULL, NULL, 16) == 0);
    assert(llgo_gc_target_markers("bad", NULL, NULL, 16) == 0);

    if (argc == 2 && strcmp(argv[1], "markers") == 0) {
        llgo_gc_init_markers();
        GC_init();
        /* Calling after GC_init must be a safe no-op. */
        llgo_gc_init_markers();
        printf("parallel=%d cpus=%u\n", GC_get_parallel(), llgo_gc_cpu_count());
        return 0;
    }

    GC_init();
    if (argc == 2 && strcmp(argv[1], "host-disabled") == 0) {
        GC_set_disable_automatic_collection(1);
        llgo_gc_init_policy();
        assert(GC_get_disable_automatic_collection());
        assert(llgo_gc_set_percent(-1) == 100);
        assert(llgo_gc_set_percent(100) == -1);
        assert(GC_get_disable_automatic_collection());
        return 0;
    }
    GC_set_on_collection_event(previous_event);
    llgo_gc_init_policy();
    llgo_gc_init_policy(); /* Must not chain the callback to itself. */
    if (argc == 2) {
        assert(llgo_gc_set_percent(100) == atoi(argv[1]));
        return 0;
    }
    assert(GC_get_min_bytes_allocd() >= 4u << 20);
#if defined(_WIN32)
    const uintptr_t default_divisor = 3;
#else
    const uintptr_t default_divisor = 1;
#endif
    assert(GC_get_free_space_divisor() == default_divisor);
    for (size_t i = 0; i < 2048; ++i) {
        unsigned char *p = GC_malloc_atomic(4096);
        assert(p != NULL);
        p[0] = (unsigned char)i;
        retained[i] = p;
    }
    GC_gcollect();
    for (size_t i = 0; i < 2048; ++i)
        assert(((unsigned char *)retained[i])[0] == (unsigned char)i);
#if defined(_WIN32)
    /* A retained heap must not turn Windows' fixed floor into a growing
     * occupied-block target. The full SIMD corpus exhausts the region table
     * with that policy, even when isolated TestAdd passes. */
    size_t minimum = llgo_gc_original_min > (4u << 20)
        ? llgo_gc_original_min : (4u << 20);
    size_t actual = GC_get_min_bytes_allocd();
    if (actual != minimum)
        fprintf(stderr, "Windows GC floor changed with occupied blocks: %zu != %zu\n", actual, minimum);
    assert(actual == minimum);
#else
    assert(GC_get_min_bytes_allocd() >= 8u << 20);
#endif
    for (size_t i = 0; i < 2048; ++i)
        retained[i] = NULL;
    GC_gcollect();
    assert(llgo_gc_set_percent(-2) == 100);
    assert(GC_get_disable_automatic_collection());
    uintptr_t before = GC_get_gc_no();
    for (int i = 0; i < 8192; ++i) {
        sink = GC_malloc_atomic(4096);
        assert(sink != NULL);
    }
    assert(GC_get_gc_no() == before);
    GC_gcollect();
    assert(GC_get_gc_no() > before);
    assert(previous_callbacks > 0);
    assert(llgo_gc_set_percent(0) == -1);
    assert(!GC_get_disable_automatic_collection());
    before = GC_get_gc_no();
    for (int i = 0; i < 16384; ++i) {
        sink = GC_malloc_atomic(4096);
        assert(sink != NULL);
    }
    assert(GC_get_gc_no() > before);
    assert(llgo_gc_set_percent(100) == 0);
    GC_gcollect();
    before = GC_get_gc_no();
    assert(llgo_gc_set_percent(-1) == 100);
    assert(llgo_gc_set_percent(100) == -1);
    assert(GC_get_gc_no() == before);
    assert(llgo_gc_set_percent(101) == 100);
    assert(llgo_gc_set_percent(100) == 101);
    assert(GC_get_gc_no() == before);
    assert(llgo_gc_set_percent(25) == 100);
    assert(llgo_gc_set_percent(-1) == 25);
    assert(GC_get_free_space_divisor() == 4 * default_divisor);
    before = GC_get_gc_no();
    assert(llgo_gc_set_percent(25) == -1);
    assert(GC_get_gc_no() == before);
    assert(llgo_gc_set_percent(100) == 25);
    assert(llgo_gc_set_percent(INT32_MAX) == 100);
    /* Prime the collector's cached threshold with the large percentage. */
    for (int i = 0; i < 128; ++i)
        sink = GC_malloc_atomic(4096);
    before = GC_get_gc_no();
    assert(llgo_gc_set_percent(100) == INT32_MAX);
    assert(GC_get_gc_no() > before);
    puts("BDWGC policy: budget, overflow, callback, explicit GC and resume pass");
    return 0;
}
