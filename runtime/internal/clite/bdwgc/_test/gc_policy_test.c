#include <assert.h>
#include <stdio.h>
#include "../_wrap/gc_policy.c"

extern void GC_init(void);
extern void GC_gcollect(void);
extern uintptr_t GC_get_gc_no(void);
extern void *GC_malloc_atomic(size_t);
extern int GC_get_disable_automatic_collection(void);
extern uintptr_t GC_get_free_space_divisor(void);

static unsigned previous_callbacks;
static void previous_event(int event) { if (event == 5) ++previous_callbacks; }
static void *volatile sink;

int main(int argc, char **argv)
{
    assert(llgo_gc_budget(0, 100) == 4u << 20);
    assert(llgo_gc_budget(8u << 20, 100) == 8u << 20);
    assert(llgo_gc_budget(8u << 20, 50) == 4u << 20);
    assert(llgo_gc_budget(SIZE_MAX, 200) == SIZE_MAX);
    assert(llgo_gc_budget(SIZE_MAX, INT32_MAX) == SIZE_MAX);
    assert(llgo_gc_budget(SIZE_MAX, 100) == SIZE_MAX);
    assert(llgo_gc_budget(0, 0) == 1);

    GC_init();
    GC_set_on_collection_event(previous_event);
    llgo_gc_init_policy();
    llgo_gc_init_policy(); /* Must not chain the callback to itself. */
    if (argc == 2) {
        assert(llgo_gc_set_percent(100) == atoi(argv[1]));
        return 0;
    }
    assert(GC_get_min_bytes_allocd() >= 4u << 20);
    assert(GC_get_free_space_divisor() == 1);
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
