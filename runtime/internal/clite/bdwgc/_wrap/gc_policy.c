/* Native BDWGC pacing. No Go code or allocation runs under the collector lock. */
#if defined(_WIN32)
#define _CRT_SECURE_NO_WARNINGS 1
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#else
#include <sched.h>
#endif
#include <stdint.h>
#include <stddef.h>
#include <stdlib.h>
#include <errno.h>
#include <stdatomic.h>

/* This stable prefix is deliberately smaller than GC_prof_stats_s. The API
 * accepts the caller's size, including clients built against older headers. */
typedef struct {
    uintptr_t heap_size;
    uintptr_t free_bytes;
    uintptr_t unmapped_bytes;
} llgo_gc_stats;

typedef void (*llgo_gc_event_fn)(int);
extern void GC_set_on_collection_event(llgo_gc_event_fn);
extern llgo_gc_event_fn GC_get_on_collection_event(void);
extern size_t GC_get_prof_stats_unsafe(llgo_gc_stats *, size_t);
extern void *GC_call_with_alloc_lock(void *(*)(void *), void *);
extern void GC_set_min_bytes_allocd(size_t);
extern size_t GC_get_min_bytes_allocd(void);
extern void GC_set_free_space_divisor(uintptr_t);
extern uintptr_t GC_get_free_space_divisor(void);
extern void GC_set_disable_automatic_collection(int);
extern int GC_get_disable_automatic_collection(void);
extern uintptr_t GC_get_bytes_since_gc(void);
extern void GC_gcollect(void);

static int32_t llgo_gc_percent = 100;
static int32_t llgo_gc_budget_percent = 100;
static int llgo_gc_host_automatic_disabled;
static size_t llgo_gc_original_min;
static llgo_gc_event_fn llgo_gc_previous_event;
static atomic_flag llgo_gc_policy_lock = ATOMIC_FLAG_INIT;
static atomic_int llgo_gc_policy_initialized;

static void llgo_gc_yield(void)
{
#if defined(_WIN32)
    Sleep(0);
#else
    sched_yield();
#endif
}

static size_t llgo_gc_budget(size_t live, int32_t percent)
{
    size_t base = live > (4u << 20) ? live : (4u << 20);
    size_t result;
    if (percent < 0)
        return SIZE_MAX;
    if (percent == 0)
        return 1;
    /* Divide first, preserving the remainder, without overflowing size_t. */
    if (base / 100 > SIZE_MAX / (uint32_t)percent)
        return SIZE_MAX;
    result = (base / 100) * (uint32_t)percent;
    uint64_t remainder = (uint64_t)(base % 100) * (uint32_t)percent / 100;
    return remainder > SIZE_MAX - result ? SIZE_MAX : result + (size_t)remainder;
}

/* Both the collection callback and call_with_alloc_lock hold the GC lock.
 * Use the unsynchronized statistics API and setter here: the synchronized
 * variants would deadlock. Block occupancy is a conservative live estimate,
 * not an exact object count; BDWGC's own scan-cost budget is still a floor. */
static size_t llgo_gc_update_budget(void)
{
    if (llgo_gc_original_min == 0)
        llgo_gc_original_min = GC_get_min_bytes_allocd();
    size_t live = 0;
#if !defined(_WIN32)
    llgo_gc_stats stats = {0, 0, 0};
    size_t filled = GC_get_prof_stats_unsafe(&stats, sizeof(stats));
    live = filled >= 2 * sizeof(uintptr_t) && stats.heap_size > stats.free_bytes
        ? stats.heap_size - stats.free_bytes : 0;
#endif
    /* Occupied blocks include fragmentation and deferred sweep. Windows'
     * bounded VirtualAlloc-region table cannot tolerate using them as a live
     * heap target: retain the fixed minimum and let BDWGC pace larger heaps. */
    /* Preserve the last active budget while automatic GC is disabled. A
     * disable/restore pair must not invalidate the collector's cached target. */
    size_t budget = llgo_gc_budget(live, llgo_gc_budget_percent);
    /* Leave room for the collector's expansion arithmetic on 32-bit hosts. */
    if (budget > SIZE_MAX / 2)
        budget = SIZE_MAX / 2;
    /* Scale BDWGC's scan-work estimate as well as the heap-only floor, so
     * the fixed cost of scanning roots participates in the budget. */
    uint32_t percent = (uint32_t)llgo_gc_budget_percent;
#if defined(_WIN32)
    /* Keep the collector's normal expansion scale at GOGC=100 on Windows. */
    const uint32_t scan_percent = 300;
#else
    const uint32_t scan_percent = 100;
#endif
    uintptr_t divisor = percent == 0 ? UINTPTR_MAX : (scan_percent / percent + (scan_percent % percent != 0));
    GC_set_free_space_divisor(divisor > 0 ? divisor : 1);
    budget = budget > llgo_gc_original_min ? budget : llgo_gc_original_min;
    GC_set_min_bytes_allocd(budget);
    return budget;
}

static void llgo_gc_collection_event(int event)
{
    if (llgo_gc_previous_event != NULL)
        llgo_gc_previous_event(event);
    if (event == 5) /* GC_EVENT_END, a stable public enum value. */
        llgo_gc_update_budget();
}

typedef struct {
    int32_t requested;
    int32_t previous;
    int refresh;
} llgo_gc_percent_change;

static void *llgo_gc_change_percent(void *raw)
{
    llgo_gc_percent_change *change = raw;
    int32_t previous_budget = llgo_gc_budget_percent;
    uintptr_t previous_divisor = GC_get_free_space_divisor();
    change->previous = llgo_gc_percent;
    llgo_gc_percent = change->requested;
    if (change->requested >= 0)
        llgo_gc_budget_percent = change->requested;
    size_t budget = llgo_gc_update_budget();
    /* Small reductions can retain the preceding estimate for one cycle.
     * Refresh a substantially lower target, or one already exceeded, rather
     * than allowing a formerly enormous cached budget to persist indefinitely. */
    change->refresh = change->requested >= 0 && change->requested < previous_budget &&
        (GC_get_free_space_divisor() > previous_divisor ||
         (uint64_t)previous_budget > 2 * (uint64_t)change->requested ||
         GC_get_bytes_since_gc() >= budget);
    return NULL;
}

int32_t llgo_gc_set_percent(int32_t percent)
{
    llgo_gc_percent_change change = {percent < 0 ? -1 : percent, 0, 0};
    /* Serialize the two collector operations, including concurrent callers.
     * Collection callbacks never acquire this outer lock. */
    while (atomic_flag_test_and_set_explicit(&llgo_gc_policy_lock, memory_order_acquire))
        llgo_gc_yield();
    GC_call_with_alloc_lock(llgo_gc_change_percent, &change);
    GC_set_disable_automatic_collection(llgo_gc_host_automatic_disabled || percent < 0);
    atomic_flag_clear_explicit(&llgo_gc_policy_lock, memory_order_release);
    /* BDWGC caches its threshold until the next collection. A lower target
     * must not remain stuck behind an arbitrarily large previous budget.
     * Collect outside both locks: finalizers can reenter this API. Startup
     * configuration does not force a collection during runtime init. */
    if (atomic_load(&llgo_gc_policy_initialized) == 2 && change.refresh)
        GC_gcollect();
    return change.previous;
}

void llgo_gc_init_policy(void)
{
    int expected = 0;
    if (!atomic_compare_exchange_strong(&llgo_gc_policy_initialized, &expected, 1)) {
        while (atomic_load(&llgo_gc_policy_initialized) != 2)
            llgo_gc_yield();
        return;
    }
    const char *value = getenv("GOGC");
    int32_t percent = 100;
    if (value != NULL) {
        if (value[0] == 'o' && value[1] == 'f' && value[2] == 'f' && value[3] == '\0') {
            percent = -1;
        } else {
            char *end;
            errno = 0;
            long number = strtol(value, &end, 10);
            if ((value[0] == '+' || value[0] == '-' || (value[0] >= '0' && value[0] <= '9')) &&
                errno == 0 && end != value && *end == '\0' && number >= INT32_MIN && number <= INT32_MAX)
                percent = (int32_t)number;
        }
    }
    llgo_gc_host_automatic_disabled = GC_get_disable_automatic_collection();
    llgo_gc_previous_event = GC_get_on_collection_event();
    GC_set_on_collection_event(llgo_gc_collection_event);
    llgo_gc_set_percent(percent);
    atomic_store(&llgo_gc_policy_initialized, 2);
}
