/* Native BDWGC pacing. No Go code or allocation runs under the collector lock. */
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
extern void GC_set_disable_automatic_collection(int);
extern void GC_gcollect(void);

static int32_t llgo_gc_percent = 100;
static size_t llgo_gc_original_min;
static llgo_gc_event_fn llgo_gc_previous_event;
static atomic_flag llgo_gc_policy_lock = ATOMIC_FLAG_INIT;
static atomic_int llgo_gc_policy_initialized;

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
static void *llgo_gc_update_budget(void *unused)
{
    llgo_gc_stats stats = {0, 0, 0};
    (void)unused;
    if (llgo_gc_original_min == 0)
        llgo_gc_original_min = GC_get_min_bytes_allocd();
    size_t filled = GC_get_prof_stats_unsafe(&stats, sizeof(stats));
    size_t live = filled >= 2 * sizeof(uintptr_t) && stats.heap_size > stats.free_bytes
        ? stats.heap_size - stats.free_bytes : 0;
    /* Do not install SIZE_MAX while automatic GC is disabled. BDWGC caches
     * this minimum until its next collection; that would prevent resuming. */
    size_t budget = llgo_gc_budget(live, llgo_gc_percent < 0 ? 100 : llgo_gc_percent);
    /* Leave room for the collector's expansion arithmetic on 32-bit hosts. */
    if (budget > SIZE_MAX / 2)
        budget = SIZE_MAX / 2;
    /* BDWGC normally permits roughly one third of its estimated scan work
     * in allocations per collection. GOGC=100 permits the full estimate,
     * including roots; a heap-only floor misses this fixed scanning cost. */
    uint32_t percent = llgo_gc_percent < 0 ? 100 : (uint32_t)llgo_gc_percent;
    uintptr_t divisor = percent == 0 ? UINTPTR_MAX : (100u / percent + (100u % percent != 0));
    GC_set_free_space_divisor(divisor > 0 ? divisor : 1);
    GC_set_min_bytes_allocd(budget > llgo_gc_original_min ? budget : llgo_gc_original_min);
    return NULL;
}

static void llgo_gc_collection_event(int event)
{
    if (llgo_gc_previous_event != NULL)
        llgo_gc_previous_event(event);
    if (event == 5) /* GC_EVENT_END, a stable public enum value. */
        llgo_gc_update_budget(NULL);
}

static void *llgo_gc_change_percent(void *raw)
{
    int32_t *percent = raw;
    int32_t previous = llgo_gc_percent;
    llgo_gc_percent = *percent < 0 ? -1 : *percent;
    *percent = previous;
    llgo_gc_update_budget(NULL);
    return NULL;
}

int32_t llgo_gc_set_percent(int32_t percent)
{
    int32_t requested = percent < 0 ? -1 : percent;
    int disable = percent < 0;
    /* Serialize the two collector operations, including concurrent callers.
     * Collection callbacks never acquire this outer lock. */
    while (atomic_flag_test_and_set_explicit(&llgo_gc_policy_lock, memory_order_acquire)) {}
    GC_call_with_alloc_lock(llgo_gc_change_percent, &percent);
    GC_set_disable_automatic_collection(disable);
    atomic_flag_clear_explicit(&llgo_gc_policy_lock, memory_order_release);
    /* BDWGC caches its threshold until the next collection. A lower target
     * must not remain stuck behind an arbitrarily large previous budget.
     * Collect outside both locks: finalizers can reenter this API. Startup
     * configuration does not force a collection during runtime init. */
    if (atomic_load(&llgo_gc_policy_initialized) == 2 && requested >= 0 &&
        (percent < 0 || requested < percent))
        GC_gcollect();
    return percent;
}

void llgo_gc_init_policy(void)
{
    int expected = 0;
    if (!atomic_compare_exchange_strong(&llgo_gc_policy_initialized, &expected, 1)) {
        while (atomic_load(&llgo_gc_policy_initialized) != 2) {}
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
    llgo_gc_previous_event = GC_get_on_collection_event();
    GC_set_on_collection_event(llgo_gc_collection_event);
    llgo_gc_set_percent(percent);
    atomic_store(&llgo_gc_policy_initialized, 2);
}
