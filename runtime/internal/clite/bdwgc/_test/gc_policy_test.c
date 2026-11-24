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

static unsigned previous_callbacks;
static void previous_event(int event) { if (event == 5) ++previous_callbacks; }
static void *volatile sink;
static void *volatile retained[2048];

static void assert_all_zero(const unsigned char *p, size_t n)
{
    for (size_t i = 0; i < n; ++i)
        assert(p[i] == 0);
}

static void test_zero_atomic_large(void)
{
    llgo_gc_zero_atomic_large(NULL, 0);
    llgo_gc_zero_atomic_large(NULL, 131072);

    const size_t sizes[] = {1, 4096, 65536, 70000, 131072, 1114112};
    for (size_t s = 0; s < sizeof(sizes) / sizeof(sizes[0]); ++s) {
        size_t n = sizes[s];
        unsigned char *p = GC_malloc_atomic(n);
        assert(p != NULL);
        memset(p, 0xA5, n);
        llgo_gc_zero_atomic_large(p, n);
        assert_all_zero(p, n);

        /* Sparse dirty head, middle, and tail, then clear again. */
        p[0] = 0x5A;
        p[n / 2] = 0x3C;
        p[n - 1] = 0xC3;
        llgo_gc_zero_atomic_large(p, n);
        assert_all_zero(p, n);
    }

#if defined(__APPLE__)
    size_t ps = llgo_gc_get_page_size();
    assert(ps > 0 && (ps & (ps - 1)) == 0);
    size_t big = 1114112;
    /* Allocate a fresh block so its interior pages have not been touched yet. */
    unsigned char *fresh = GC_malloc_atomic(big);
    assert(fresh != NULL);
    llgo_gc_zero_atomic_large(fresh, big);
    uintptr_t addr = (uintptr_t)fresh;
    uintptr_t mask = ps - 1;
    uintptr_t start = (addr + mask) & ~mask;
    uintptr_t end = (addr + big) & ~mask;
    size_t npages = (size_t)((end - start) / ps);
    assert(npages >= 64);
    char *vec = calloc(npages, 1);
    assert(vec != NULL);
    assert(mincore((const void *)start, npages * ps, vec) == 0);
    size_t resident = 0;
    for (size_t i = 0; i < npages; ++i) {
        if (vec[i] & MINCORE_INCORE)
            ++resident;
    }
    /* Interior pages must remain unmaterialized rather than eagerly faulted. */
    assert(resident < npages / 2);
    assert_all_zero(fresh, big);

    /* Also exercise the non-MADV_ZERO fallback path explicitly. */
    unsigned char *fb = GC_malloc_atomic(big);
    assert(fb != NULL);
    uintptr_t fb_addr = (uintptr_t)fb;
    uintptr_t fb_start = (fb_addr + mask) & ~mask;
    uintptr_t fb_end = (fb_addr + big) & ~mask;
    size_t fb_span = (size_t)(fb_end - fb_start);
    assert(llgo_gc_zero_pages_darwin_fallback(fb_start, fb_span, ps) == 1);
    if (fb_start > fb_addr)
        memset(fb, 0, (size_t)(fb_start - fb_addr));
    if (fb_addr + big > fb_end)
        memset((void *)fb_end, 0, (size_t)(fb_addr + big - fb_end));
    size_t fb_pages = fb_span / ps;
    assert(fb_pages <= npages + 1);
    free(vec);
    vec = calloc(fb_pages, 1);
    assert(vec != NULL);
    assert(mincore((const void *)fb_start, fb_span, vec) == 0);
    resident = 0;
    for (size_t i = 0; i < fb_pages; ++i) {
        if (vec[i] & MINCORE_INCORE)
            ++resident;
    }
    assert(resident < fb_pages / 2);
    assert_all_zero(fb, big);

    /* Once the middle page is resident, the fallback defers to memset. */
    memset(fb, 0x7E, big);
    assert(llgo_gc_zero_pages_darwin_fallback(fb_start, fb_span, ps) == 0);
    free(vec);
#endif
}

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
    if (argc == 2 && strcmp(argv[1], "lazy-zero") == 0) {
        test_zero_atomic_large();
        return 0;
    }
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
