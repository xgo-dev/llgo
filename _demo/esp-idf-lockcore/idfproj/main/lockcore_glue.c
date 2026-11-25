/* Core-affinity shim behind runtime.LockCore (esp32s31-idf): the STRONG
 * llgoLockCore that overrides the runtime's weak default
 * (runtime/internal/runtime/_wrap/lockcore_idf.c, which returns 1 = "not
 * provided"). The whole-archive main component keeps this object in the
 * link unconditionally, so the strong definition wins.
 *
 * The IDF kernel linked into S31 firmware pins a task once, at creation:
 * xTaskCreatePinnedToCore writes TCB.xCoreID, and no kernel API writes it
 * afterwards - vTaskCoreAffinitySet belongs to the Amazon SMP kernel, whose
 * Kconfig entry is `depends on IDF_TARGET_ESP32`, so an S31 build never links
 * it (verified against espressif/idf:latest = v6.3.0). This file is therefore
 * the pin: it writes the kernel-internal field of the calling task, and the
 * scheduler's affinity check (taskIS_AFFINITY_COMPATIBLE, tasks.c) moves the
 * task at the next scheduling point.
 *
 * The offset below is kernel knowledge that no public header gives out, so
 * every write is guarded in three layers:
 *
 *   1. Version gate: this file refuses to compile against any other IDF
 *      release; the offset is only meaningful for the pinned one.
 *   2. Provenance: the offset was extracted from the DWARF of this exact
 *      build's tasks.c object - DW_AT_data_member_location 68 on the xCoreID
 *      member at tasks.c:432 - not guessed (recipe in ../README.md).
 *   3. Self-check: before writing, the bytes at the offset must read back as
 *      the same core id the public xTaskGetCoreID getter returns; after
 *      writing, the getter must confirm the new id, otherwise the write is
 *      reverted and refused. A layout drift therefore makes llgoLockCore
 *      return nonzero - which runtime.LockCore maps to an error - instead of
 *      corrupting the TCB.
 *
 * The store itself is one aligned 32-bit write, atomic on RV32; the field has
 * exactly one writer (creation, then this shim), and plain scheduler-side
 * reads cannot observe a torn value. The known race boundary: the task
 * mutates its own pin while running; scheduler-side affinity reads happen at
 * selection points, so a selection that already passed the check on the old
 * core can run the task there one more quantum before the next scheduling
 * point migrates it - eventual consistency, never memory unsafety.
 */

#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "esp_cpu.h"

#if ESP_IDF_VERSION != ESP_IDF_VERSION_VAL(6, 3, 0)
#error "lockcore_glue.c pins the IDF kernel's internal TCB layout; \
re-extract LLGO_TCB_XCOREID_OFFSET from the new build's tasks.c DWARF \
before building against a different IDF version"
#endif

/* offsetof(tskTCB, xCoreID) in components/freertos/FreeRTOS-Kernel/tasks.c
 * (the field only exists when configNUMBER_OF_CORES > 1), extracted from this
 * build's tasks.c object DWARF. */
#define LLGO_TCB_XCOREID_OFFSET 68

_Static_assert(LLGO_TCB_XCOREID_OFFSET % 4 == 0,
               "xCoreID offset must keep the BaseType_t aligned");

int llgoCurrentCore(void)
{
    /* esp_cpu_get_core_id is an inline reading the executing hart - there is
     * no out-of-line symbol to linkname, hence this wrapper. */
    return esp_cpu_get_core_id();
}

int llgoLockCore(int core)
{
    TaskHandle_t handle = xTaskGetCurrentTaskHandle();
    if (handle == NULL) {
        return -1;
    }

    volatile BaseType_t *field =
        (volatile BaseType_t *)((char *)handle + LLGO_TCB_XCOREID_OFFSET);

    /* The kernel's own getter reads the true field; this shim's view of the
     * layout must agree with it before anything is written. */
    BaseType_t old = xTaskGetCoreID(handle);
    if (old != *field) {
        return -2;
    }

    *field = (BaseType_t)core;
    if (xTaskGetCoreID(handle) != (BaseType_t)core) {
        *field = old; /* the write landed elsewhere: undo, refuse */
        return -3;
    }
    return 0;
}
